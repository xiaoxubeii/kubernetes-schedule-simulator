package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/xiaoxubeii/kubernetes-schedule-simulator/pkg/scheduler/core"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/kubernetes/pkg/features"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
)

var (
	MB  int64 = 1024 * 1024
	GB  int64 = MB * 1024
	CPU int64 = 1000
)

func init() {
	utilfeature.DefaultMutableFeatureGate.Set(fmt.Sprintf("%s=false,%s=false", features.VolumeScheduling, features.TaintNodesByCondition))
}

func main() {
	flag.Parse()
	defer glog.Flush()

	// nodes, pods, err := anaCheckPoint()
	nodes, pods, err := createSampleResource()
	if err != nil {
		glog.Fatalf("fail to get pods or nodes: %v", err)
	}
	s := core.NewSimulator(nodes, pods)

	start := time.Now()
	r, fail := s.Schedule()
	end := time.Now()
	glog.Infof("total time: %v\n", end.Sub(start))
	glog.Infof("success: %v, fail: %v", len(r), len(fail))
}

func anaCheckPoint() ([]*v1.Node, []*v1.Pod, error) {
	pods, err := getPodsCheckPoint()
	if err != nil {
		return nil, nil, err
	}
	nodes, err := getNodeCheckPoint()
	if err != nil {
		return nil, nil, err
	}

	return nodes, pods, nil
}

func getPodsCheckPoint() ([]*v1.Pod, error) {
	d, err := ioutil.ReadFile("pods.json")
	if err != nil {
		return nil, err
	}
	pods := []v1.Pod{}
	err = json.Unmarshal(d, &pods)
	if err != nil {
		return nil, err
	}
	newPods := []*v1.Pod{}
	for i, _ := range pods {
		newPods = append(newPods, &pods[i])
	}
	return newPods, nil
}

func getNodeCheckPoint() ([]*v1.Node, error) {
	d, err := ioutil.ReadFile("nodes.json")
	if err != nil {
		return nil, err
	}
	nodes := []v1.Node{}
	err = json.Unmarshal(d, &nodes)
	if err != nil {
		return nil, err
	}
	newNodes := []*v1.Node{}
	for i, _ := range nodes {
		newNodes = append(newNodes, &nodes[i])
	}
	return newNodes, nil
}

func createSampleResource() ([]*v1.Node, []*v1.Pod, error) {
	podRes := schedulercache.Resource{MilliCPU: 1 * CPU, Memory: 1 * GB}
	pods := createSamplePods(100, podRes)
	nodeRes := schedulercache.Resource{MilliCPU: 64 * CPU, Memory: 64 * GB, AllowedPodNumber: 1000}
	nodes := createSampleNodes(1, nodeRes)
	return nodes, pods, nil
}

func createSamplePods(podNum int, podRes schedulercache.Resource) []*v1.Pod {
	pods := []*v1.Pod{}
	for i := 0; i < podNum; i++ {
		pod := newSamplePod(podRes)
		pod.UID = types.UID(uuid.New().String())
		pods = append(pods, pod)
	}
	return pods
}

func createSampleNodes(nodeNum int, nodeRes schedulercache.Resource) []*v1.Node {
	nodes := []*v1.Node{}
	for i := 0; i < nodeNum; i++ {
		node := newSampleNode(nodeRes)
		node.UID = types.UID(uuid.New().String())
		node.Name = string(node.UID)
		nodes = append(nodes, node)
	}
	return nodes
}

func newSamplePod(usage ...schedulercache.Resource) *v1.Pod {
	containers := []v1.Container{}
	for _, req := range usage {
		containers = append(containers, v1.Container{
			Resources: v1.ResourceRequirements{Requests: req.ResourceList()},
		})
	}
	return &v1.Pod{
		Spec: v1.PodSpec{
			Containers: containers,
		},
	}
}

func newSampleNode(usage schedulercache.Resource) *v1.Node {
	return &v1.Node{
		Status: v1.NodeStatus{
			Capacity:    usage.ResourceList(),
			Allocatable: usage.ResourceList(),
		}}
}

package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/xiaoxubeii/kubernetes-schedule-simulator/pkg/framework"
	"github.com/xiaoxubeii/kubernetes-schedule-simulator/pkg/scheduler"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	schedapp "k8s.io/kubernetes/cmd/kube-scheduler/app"
	"k8s.io/kubernetes/pkg/apis/componentconfig"
	"k8s.io/kubernetes/pkg/scheduler/schedulercache"
)

var (
	MB  int64 = 1024 * 1024
	GB  int64 = MB * 1024
	CPU int64 = 1000
)

func main() {
	flag.Parse()
	defer glog.Flush()
	pods := make([]v1.Pod, 0)
	nodes := make([]v1.Node, 0)
	newPods := make([]*v1.Pod, 0)

	// TODO
	syncCheckPoints := true
	namespace := ""
	// podA := schedulercache.Resource{MilliCPU: 9 * CPU, Memory: 43 * GB}
	// podBRes := schedulercache.Resource{MilliCPU: 5 * CPU, Memory: 18 * GB}
	// podB := createSamplePods(100, podBRes)
	podARes := schedulercache.Resource{MilliCPU: 9 * CPU, Memory: 43 * GB}
	podA := createSamplePods(100, podARes)
	newPods = append(newPods, podA...)

	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	if syncCheckPoints {
		// use the current context in kubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			glog.Fatalf("Failed to get config: %v", err)
		}
		// create the clientset
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			glog.Fatalf("Failed to create clientset: %v", err)
		}

		// nodes, pods, _ := createSampleResource()
		glog.V(1).Infoln("Begin to get checkpoints")
		pods, nodes, err = getCheckpoints(clientset, namespace)
		if err != nil {
			glog.Fatalf("Failed to get checkpoints: %v", err)
		}
	}

	provider := "TalkintDataProvider"
	conf := &componentconfig.KubeSchedulerConfiguration{
		SchedulerName:                  "test",
		AlgorithmSource:                componentconfig.SchedulerAlgorithmSource{Provider: &provider},
		HardPodAffinitySymmetricWeight: 10,
	}
	newScheduler, err := schedapp.NewSchedulerServer(conf, "localhost")
	if err != nil {
		glog.Fatalf("Failed to create scheduler server: %v", err)
	}

	report, err := runSimulator(newScheduler, pods, nodes, newPods)
	if err != nil {
		glog.Fatalf("Failed to start scheduler simulator: %v", err)
	}

	framework.ClusterCapacityReviewPrint(report)
}

func getCheckpoints(clientset *kubernetes.Clientset, namespace string) ([]v1.Pod, []v1.Node, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{FieldSelector: "status.phase=Running"})
	if err != nil {
		return nil, nil, err
	}
	glog.V(1).Infof("Get scheduled pods num: %v", len(pods.Items))

	// TODO
	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	glog.V(1).Infof("Get nodes num: %v", len(nodes.Items))

	return pods.Items, nodes.Items, err
}

func runSimulator(server *schedapp.SchedulerServer, pods []v1.Pod, nodes []v1.Node, newPods []*v1.Pod) (*framework.GeneralReview, error) {
	cc, err := scheduler.New(server, newPods, pods, nodes)
	if err != nil {
		return nil, err
	}

	// TODO (avesh): Enable when support for multiple schedulers is implemented.
	/*for i := 0; i < len(s.Schedulers); i++ {
		if err = cc.AddScheduler(s.Schedulers[i]); err != nil {
			return nil, err
		}
	}*/

	glog.V(1).Infoln("Begin to schedule")
	err = cc.Run()
	if err != nil {
		return nil, err
	}

	report := cc.Report()
	return report, nil
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
	pods := createSamplePods(2, podRes)
	nodeRes := schedulercache.Resource{MilliCPU: 1 * CPU, Memory: 1 * GB, AllowedPodNumber: 1000}
	nodes := createSampleNodes(2, nodeRes)
	return nodes, pods, nil
}

func createSamplePods(podNum int, podRes schedulercache.Resource) []*v1.Pod {
	pods := []*v1.Pod{}
	for i := 0; i < podNum; i++ {
		pod := newSamplePod(podRes)
		pod.UID = types.UID(uuid.New().String())
		pod.Name = string(pod.UID)
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

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

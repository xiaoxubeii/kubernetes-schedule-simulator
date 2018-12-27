package core

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	"github.com/xiaoxubeii/kubernetes-schedule-simulator/pkg/scheduler/cache"
	"github.com/xiaoxubeii/kubernetes-schedule-simulator/pkg/scheduler/defaults"
	"github.com/xiaoxubeii/kubernetes-schedule-simulator/pkg/scheduler/factory"
	"github.com/xiaoxubeii/kubernetes-schedule-simulator/pkg/scheduler/queue"
	"k8s.io/kubernetes/pkg/scheduler/algorithm/predicates"
	"k8s.io/kubernetes/pkg/scheduler/core/equivalence"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/pkg/scheduler/algorithm"
	kfactory "k8s.io/kubernetes/pkg/scheduler/factory"
)

type Simulator struct {
	scheduler           algorithm.ScheduleAlgorithm
	nodeLister          algorithm.NodeLister
	nodes               []*v1.Node
	pods                []*v1.Pod
	schedulecache       cache.Cache
	equivalencePodCache *equivalence.Cache
}

func (s *Simulator) NextPod() *v1.Pod {
	if len(s.pods) > 0 {
		p := s.pods[0]
		s.pods = s.pods[1:]
		return p
	}
	return nil
}

func (s *Simulator) Bind(nodeName string, pod *v1.Pod) {
	pod.Spec.NodeName = nodeName
	s.schedulecache.AddPod(pod)
}

func (s *Simulator) addNodeToCache(obj interface{}) {
	node, ok := obj.(*v1.Node)
	if !ok {
		glog.Errorf("cannot convert to *v1.Node: %v", obj)
		return
	}

	s.equivalencePodCache.GetNodeCache(node.GetName())

	if err := s.schedulecache.AddNode(node); err != nil {
		glog.Errorf("scheduler cache AddNode failed: %v", err)
	}
}

func (s *Simulator) Schedule() (r map[string]string, fail map[string]string) {
	r = make(map[string]string)
	fail = make(map[string]string)
	for {
		pod := s.NextPod()
		if pod != nil {
			glog.V(1).Infof("pod: %v", pod)
			host, err := s.schedule(pod)
			if err != nil {
				fail[string(pod.UID)] = fmt.Sprintf("%v", err)
				continue
			}
			s.Bind(host, pod)
			r[string(pod.UID)] = host
		} else {
			return
		}
	}
}

func (s *Simulator) schedule(pod *v1.Pod) (string, error) {
	host, err := s.scheduler.Schedule(pod, s.nodeLister)
	return host, err
}

type fakeNodeLister struct {
	nodes []*v1.Node
}

func (n *fakeNodeLister) List() ([]*v1.Node, error) {
	return n.nodes, nil
}

type fakeServiceLister struct {
	servers []*v1.Service
}

func (f *fakeServiceLister) List(labels.Selector) ([]*v1.Service, error) {
	return nil, nil
}

func (f *fakeServiceLister) GetPodServices(*v1.Pod) ([]*v1.Service, error) {
	return nil, nil
}

func NewSimulator(nodes []*v1.Node, pods []*v1.Pod) *Simulator {
	defaults.ApplyFeatureGates()
	provider, _ := factory.GetAlgorithmProvider(kfactory.DefaultProvider)
	stop := make(chan struct{})
	schedulerCache := cache.New(30*time.Second, stop)
	podQueue := queue.NewSchedulingQueue(stop)
	ecache := equivalence.NewCache(predicates.Ordering())

	config := factory.Config{
		NodeLister:        &fakeNodeLister{nodes},
		ServiceLister:     &fakeServiceLister{},
		ControllerLister:  &algorithm.EmptyControllerLister{},
		ReplicaSetLister:  &algorithm.EmptyReplicaSetLister{},
		StatefulSetLister: &algorithm.EmptyStatefulSetLister{},
	}
	predicateFuncs, _ := config.GetPredicates(provider.FitPredicateKeys)
	priorityConfigs, _ := config.GetPriorityFunctionConfigs(provider.PriorityFunctionKeys)
	predicateMetaProducer, _ := config.GetPredicateMetadataProducer()
	priorityMetaProducer, _ := config.GetPriorityMetadataProducer()
	algo := NewGenericScheduler(
		schedulerCache,
		ecache,
		podQueue,
		predicateFuncs,
		predicateMetaProducer,
		priorityConfigs,
		priorityMetaProducer,
		nil,
		nil,
		nil,
		nil,
		nil,
		true,
		true,
		100,
	)

	s := &Simulator{
		scheduler: algo, nodes: nodes, pods: pods, nodeLister: config.NodeLister,
		schedulecache:       schedulerCache,
		equivalencePodCache: ecache,
	}

	for _, n := range nodes {
		s.addNodeToCache(n)
	}

	return s

}

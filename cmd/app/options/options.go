/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package options

import (
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"github.com/google/uuid"
	kapi "github.com/xiaoxubeii/kubernetes-schedule-simulator/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	schedapp "k8s.io/kubernetes/cmd/kube-scheduler/app"
	"k8s.io/kubernetes/pkg/apis/componentconfig"
)

type K8SSchedulerSimulatorConf struct {
	Options   *ClusterCapacityOptions
	Scheduler *schedapp.SchedulerServer
}

type ClusterCapacityOptions struct {
	AlgorithmProvider string
	Kubeconfig        string
	PodSpecFile       string
	Namespace         string
}

func NewClusterCapacityConfig(opt *ClusterCapacityOptions) *K8SSchedulerSimulatorConf {
	conf := &componentconfig.KubeSchedulerConfiguration{
		SchedulerName:                  "TD-Scheduler",
		AlgorithmSource:                componentconfig.SchedulerAlgorithmSource{Provider: &opt.AlgorithmProvider},
		HardPodAffinitySymmetricWeight: 10,
	}
	newScheduler, err := schedapp.NewSchedulerServer(conf, "localhost")
	if err != nil {
		glog.Fatalf("Failed to create scheduler server: %v", err)
	}
	return &K8SSchedulerSimulatorConf{
		Options:   opt,
		Scheduler: newScheduler,
	}
}

func NewClusterCapacityOptions() *ClusterCapacityOptions {
	return &ClusterCapacityOptions{}
}

func (s *ClusterCapacityOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.Kubeconfig, "kubeconfig", s.Kubeconfig, "Path to the kubeconfig file to use for the analysis.")
	fs.StringVar(&s.AlgorithmProvider, "algorithmprovider", "DefaultProvider", "Kubernetes scheduler algorithm provider.")
	fs.StringVar(&s.PodSpecFile, "podspec", s.PodSpecFile, "Path to JSON or YAML file containing pod definition.")
}

func (s *K8SSchedulerSimulatorConf) ParseSimulationPod() ([]*v1.Pod, error) {
	spec, err := os.Open(s.Options.PodSpecFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to open config file: %v", err)
	}

	decoder := yaml.NewYAMLOrJSONDecoder(spec, 4096)
	versionedPod := make([]kapi.SimulationPod, 0)
	err = decoder.Decode(&versionedPod)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode config file: %v", err)
	}

	pods := make([]*v1.Pod, 0)
	for _, p := range versionedPod {
		for i := 0; i < p.Num; i++ {
			pod := p.Pod.DeepCopy()
			pod.UID = types.UID(uuid.New().String())
			pod.Name = string(pod.UID)
			pod.Labels = map[string]string{"SimulationName": p.Name}
			pod.Namespace = s.Options.Namespace
			pods = append(pods, pod)
		}
	}

	return pods, nil
}

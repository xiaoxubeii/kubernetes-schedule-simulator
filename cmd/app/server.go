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

package app

import (
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/xiaoxubeii/kubernetes-schedule-simulator/cmd/app/options"
	"github.com/xiaoxubeii/kubernetes-schedule-simulator/pkg/framework"
	"github.com/xiaoxubeii/kubernetes-schedule-simulator/pkg/scheduler"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	schedapp "k8s.io/kubernetes/cmd/kube-scheduler/app"
)

func NewClusterCapacityCommand() *cobra.Command {
	opt := options.NewClusterCapacityOptions()
	cmd := &cobra.Command{
		Use:   "k8s-scheduler-simulator --kubeconfig KUBECONFIG --podspec PODSPEC --algorithmprovider",
		Short: "k8s-scheduler-simulator is used for simulating scheduling of one or multiple pods",
		Run: func(cmd *cobra.Command, args []string) {
			err := Validate(opt)
			if err != nil {
				fmt.Println(err)
				cmd.Help()
				return
			}
			err = Run(opt)
			if err != nil {
				fmt.Println(err)
			}
		},
	}
	opt.AddFlags(cmd.Flags())
	return cmd
}

func Validate(opt *options.ClusterCapacityOptions) error {
	if len(opt.PodSpecFile) == 0 {
		return fmt.Errorf("Pod spec file is missing")
	}

	_, present := os.LookupEnv("CC_INCLUSTER")
	if !present {
		if len(opt.Kubeconfig) == 0 {
			return fmt.Errorf("kubeconfig is missing")
		}
	}
	return nil
}

func Run(opt *options.ClusterCapacityOptions) error {
	conf := options.NewClusterCapacityConfig(opt)

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", opt.Kubeconfig)
	if err != nil {
		return fmt.Errorf("Failed to get config: %v", err)
	}
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("Failed to create clientset: %v", err)
	}

	pods, nodes, err := getCheckpoints(clientset, opt.Namespace)
	if err != nil {
		return fmt.Errorf("Failed to get checkpoints: %v", err)
	}

	simulationPods, err := conf.ParseSimulationPod()
	if err != nil {
		return fmt.Errorf("Failed to parse simulation pod spec: %v", err)
	}

	report, err := runSimulator(conf.Scheduler, pods, simulationPods, nodes)
	if err != nil {
		return fmt.Errorf("Failed to start scheduler simulator: %v", err)
	}

	framework.ClusterCapacityReviewPrint(report)
	return nil
}

func getCheckpoints(clientset *kubernetes.Clientset, namespace string) ([]v1.Pod, []v1.Node, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{FieldSelector: "status.phase=Running"})
	if err != nil {
		return nil, nil, err
	}
	glog.V(1).Infof("Get scheduled pods num: %v", len(pods.Items))

	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	glog.V(1).Infof("Get nodes num: %v", len(nodes.Items))

	return pods.Items, nodes.Items, err
}

func runSimulator(server *schedapp.SchedulerServer, scheduledPods []v1.Pod, simulationPods []*v1.Pod, nodes []v1.Node) (*framework.GeneralReview, error) {
	cc, err := scheduler.New(server, simulationPods, scheduledPods, nodes)
	if err != nil {
		return nil, err
	}

	glog.V(1).Infoln("Begin to schedule")
	err = cc.Run()
	if err != nil {
		return nil, err
	}

	report := cc.Report()
	return report, nil
}

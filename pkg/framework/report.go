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

package framework

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"

	//"strconv"
	"strings"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/core/v1/helper"
)

type ClusterCapacityReview struct {
	unversioned.TypeMeta
	Spec   *ClusterCapacityReviewSpec
	Status *ClusterCapacityReviewStatus
}

type GeneralReview struct {
	Review     map[string]*ClusterCapacityReview
	FailReason *ClusterCapacityReviewScheduleFailReason
}

type ClusterCapacityReviewSpec struct {
	// the pod desired for scheduling
	Pods            []*v1.Pod
	PodRequirements []*Requirements
}

type ClusterCapacityReviewStatus struct {
	CreationTimestamp time.Time
	// per node information about the scheduling simulation
	Pods          []*PodReviewResult
	ReasonSummary map[string][]*PodReviewResult
}

type PodReviewResult struct {
	PodUID    string
	Host      string
	PodName   string
	Reason    string
	Resources *Resources
	// Message string
}

type Resources struct {
	PrimaryResources v1.ResourceList
	ScalarResources  map[v1.ResourceName]int64
}

type Requirements struct {
	PodName       string
	Resources     *Resources
	NodeSelectors map[string]string
}

type ClusterCapacityReviewScheduleFailReason struct {
	FailType    string
	FailMessage string
}

func getMainFailReason(message string) *ClusterCapacityReviewScheduleFailReason {
	slicedMessage := strings.Split(message, "\n")
	colon := strings.Index(slicedMessage[0], ":")

	fail := &ClusterCapacityReviewScheduleFailReason{
		FailType:    slicedMessage[0][:colon],
		FailMessage: strings.Trim(slicedMessage[0][colon+1:], " "),
	}
	return fail
}

func getResourceRequest(pod *v1.Pod) *Resources {
	result := Resources{
		PrimaryResources: v1.ResourceList{
			v1.ResourceName(v1.ResourceCPU):       *resource.NewMilliQuantity(0, resource.DecimalSI),
			v1.ResourceName(v1.ResourceMemory):    *resource.NewQuantity(0, resource.BinarySI),
			v1.ResourceName(v1.ResourceNvidiaGPU): *resource.NewMilliQuantity(0, resource.DecimalSI),
		},
	}

	for _, container := range pod.Spec.Containers {
		for rName, rQuantity := range container.Resources.Requests {
			switch rName {
			case v1.ResourceMemory:
				rQuantity.Add(*(result.PrimaryResources.Memory()))
				result.PrimaryResources[v1.ResourceMemory] = rQuantity
			case v1.ResourceCPU:
				rQuantity.Add(*(result.PrimaryResources.Cpu()))
				result.PrimaryResources[v1.ResourceCPU] = rQuantity
			case v1.ResourceNvidiaGPU:
				rQuantity.Add(*(result.PrimaryResources.NvidiaGPU()))
				result.PrimaryResources[v1.ResourceNvidiaGPU] = rQuantity
			default:
				if helper.IsScalarResourceName(rName) {
					// Lazily allocate this map only if required.
					if result.ScalarResources == nil {
						result.ScalarResources = map[v1.ResourceName]int64{}
					}
					result.ScalarResources[rName] += rQuantity.Value()
				}
			}
		}
	}
	return &result
}

func getPodsRequirements(pods []*v1.Pod) []*Requirements {
	result := make([]*Requirements, 0)
	for _, pod := range pods {
		podRequirements := &Requirements{
			PodName:       pod.Name,
			Resources:     getResourceRequest(pod),
			NodeSelectors: pod.Spec.NodeSelector,
		}
		result = append(result, podRequirements)
	}
	return result
}

func getReviewSpec(pods []*v1.Pod) *ClusterCapacityReviewSpec {
	return &ClusterCapacityReviewSpec{
		Pods:            pods,
		PodRequirements: getPodsRequirements(pods),
	}
}

func getReviewStatus(pods []*v1.Pod) *ClusterCapacityReviewStatus {
	reasonSummary := make(map[string][]*PodReviewResult)
	prrs := make([]*PodReviewResult, 0)
	for _, p := range pods {
		prr := &PodReviewResult{PodUID: string(p.UID), PodName: p.Name, Host: p.Spec.NodeName, Reason: p.Status.Reason,
			Resources: getResourceRequest(p)}
		if _, ok := reasonSummary[prr.Reason]; !ok {
			reasonSummary[prr.Reason] = make([]*PodReviewResult, 0)
		}
		reasonSummary[prr.Reason] = append(reasonSummary[prr.Reason], prr)
		prrs = append(prrs, prr)
	}

	return &ClusterCapacityReviewStatus{CreationTimestamp: time.Now(), Pods: prrs, ReasonSummary: reasonSummary}

}

func GetReport(status Status) *GeneralReview {
	rm := make(map[string]*ClusterCapacityReview)
	rm["failed"] = &ClusterCapacityReview{Spec: getReviewSpec(status.FailedPods), Status: getReviewStatus(status.FailedPods)}
	rm["success"] = &ClusterCapacityReview{Spec: getReviewSpec(status.SuccessfulPods), Status: getReviewStatus(status.SuccessfulPods)}
	rm["scheduled"] = &ClusterCapacityReview{Spec: getReviewSpec(status.ScheduledPods), Status: getReviewStatus(status.ScheduledPods)}
	return &GeneralReview{Review: rm, FailReason: &ClusterCapacityReviewScheduleFailReason{FailType: "Stopped", FailMessage: status.StopReason}}
}

func specPrint(spec *ClusterCapacityReviewSpec) {
	for _, req := range spec.PodRequirements {
		fmt.Printf("%v pod requirements:\n", req.PodName)
		fmt.Printf("\t- CPU: %v\n", req.Resources.PrimaryResources.Cpu().String())
		fmt.Printf("\t- Memory: %v\n", req.Resources.PrimaryResources.Memory().String())
		if !req.Resources.PrimaryResources.NvidiaGPU().IsZero() {
			fmt.Printf("\t- NvidiaGPU: %v\n", req.Resources.PrimaryResources.NvidiaGPU().String())
		}
		if req.Resources.ScalarResources != nil {
			fmt.Printf("\t- ScalarResources: %v\n", req.Resources.ScalarResources)
		}

		if req.NodeSelectors != nil {
			fmt.Printf("\t- NodeSelector: %v\n", labels.SelectorFromSet(labels.Set(req.NodeSelectors)).String())
		}
		fmt.Printf("\n")
	}
}

func statusPrint(status *ClusterCapacityReviewStatus) {
	fmt.Println("Pods summary:")
	for k, v := range status.ReasonSummary {
		fmt.Printf("\t- %s: %d\n", k, len(v))
	}
}

func printHeader(title string) {
	fmt.Printf("================================= %s =================================\n", title)
}

func successPodsPrint(r *GeneralReview) {
	review := r.Review["success"]
	printHeader("Successful Pods")
	distributePodsPrint(review)
}

func distributePodsPrint(r *ClusterCapacityReview) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Requirements", "Host"})
	data := make([][]string, 0)
	for _, s := range r.Status.Pods {
		data = append(data, []string{fmt.Sprintf("CPU: %s, Memory: %s", s.Resources.PrimaryResources.Cpu(), s.Resources.PrimaryResources.Memory()), s.Host})
	}

	for _, v := range data {
		table.Append(v)
	}

	table.Render()
}

func failedPodsPrint(r *GeneralReview) {
	review := r.Review["failed"]
	printHeader("Failed Pods")
	statusPrint(review.Status)
	distributePodsPrint(review)
}

func ClusterCapacityReviewPrint(r *GeneralReview) {
	successPodsPrint(r)
	failedPodsPrint(r)
}

// capture all scheduled pods with reason why the analysis could not continue
type Status struct {
	SuccessfulPods []*v1.Pod
	FailedPods     []*v1.Pod
	ScheduledPods  []*v1.Pod
	StopReason     string
}

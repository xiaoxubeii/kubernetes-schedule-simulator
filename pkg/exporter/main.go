package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	flag.Parse()
	defer glog.Flush()
	err := exportResources()
	if err != nil {
		glog.Fatalf("fail to export resource: %v", err)
	}
}

func getClientset() (*kubernetes.Clientset, error) {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func getRawPods(clientset *kubernetes.Clientset) ([]v1.Pod, error) {
	pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

func transformedPods(clientset *kubernetes.Clientset) ([]Pod, error) {
	rawPods, err := getRawPods(clientset)
	if err != nil {
		return nil, err
	}
	pods := []Pod{}
	for _, p := range rawPods {
		pods = append(pods, Pod{
			Name:      p.Name,
			UID:       string(p.UID),
			Namespace: p.Namespace,
			NodeName:  p.Spec.NodeName,
			PodIP:     p.Status.PodIP,
		})
	}
	return pods, nil
}

type Node struct {
	Name         string
	UID          string
	CapacityCPU  float32
	CapacityCMem float32
	CapacityPods float32
}

type Pod struct {
	Name       string
	UID        string
	Namespace  string
	CreateTime string
	DeleteTime string
	NodeName   string
	PodIP      string
}

func getNodes(clientset *kubernetes.Clientset) ([]v1.Node, error) {
	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nodes.Items, nil
}

func exportResources() error {
	clientset, err := getClientset()
	if err != nil {
		return err
	}
	pods, err := getRawPods(clientset)
	if err != nil {
		return err
	}
	d, err := json.Marshal(pods)
	if err != nil {
		return err
	}
	err = writeFile("./pods.json", d)
	if err != nil {
		return err
	}

	nodes, err := getNodes(clientset)
	n, err := json.Marshal(nodes)
	if err != nil {
		return err
	}
	err = writeFile("./nodes.json", n)
	return err
}

func writeFile(path string, d []byte) error {
	return ioutil.WriteFile(path, d, 0644)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

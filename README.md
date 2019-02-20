# WARNING
This software is a purely open source for internal use and fun, I do not guarantee the quality of the software :smiling_imp:

# Kubernetes Schedule Simulator
Kubernetes Schedule Simulator aims to simulate scheduling behavior in Kubernetes. It helps to tune scheduling performance, validate schedule algorithms and confirm policy changes, in offline instead of production environment.

# Quickstart

```
$ go run pkg/main.go -v 1 -alsologtostderr
a6c86967-3fab-4005-ba2b-27587625cf47 pod requirements:
        - CPU: 1
        - Memory: 1Gi

932b4723-428b-4fa2-b795-5add19b5d7b3 pod requirements:
        - CPU: 1
        - Memory: 1Gi

The cluster can schedule 1 instance(s) of the pod a6c86967-3fab-4005-ba2b-27587625cf47.
The cluster can schedule 1 instance(s) of the pod 932b4723-428b-4fa2-b795-5add19b5d7b3.

Termination reason: fail to get next pod: No pods left

Pod distribution among nodes:
a6c86967-3fab-4005-ba2b-27587625cf47
        - 3bad8737-6d7f-4f31-9fed-840b8155f1e1: 1 instance(s)
932b4723-428b-4fa2-b795-5add19b5d7b3
        - 2f7fd10e-dc85-4782-ba26-034eff407623: 1 instance(s)
```

# Notes
This project has a deep reference to [https://github.com/kubernetes-incubator/cluster-capacity](https://github.com/kubernetes-incubator/cluster-capacity) and copied a lot of codes, PLEASE KNOW.
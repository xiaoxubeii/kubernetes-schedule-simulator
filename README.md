# WARNING
This software is a purely open source for internal use and fun, I do not guarantee the quality of the software :smiling_imp:

# Kubernetes Schedule Simulator
Kubernetes Schedule Simulator aims to simulate scheduling behavior in Kubernetes. It helps to tune scheduling performance, validate schedule algorithms and confirm policy changes, in offline instead of production environment.

# Installation

```
$ go build -o k8s-scheduler-simulator cmd/main.go
```

# Quickstart

```
$ ./k8s-scheduler-simulator --kubeconfig ~/.kube/config --podspec ./etc/pod.yaml

================================= Successful Pods =================================
+-------------------+--------------------------------------+
|   REQUIREMENTS    |                 HOST                 |
+-------------------+--------------------------------------+
| CPU: 1, Memory: 1 | test-1474.test.com |
| CPU: 1, Memory: 1 | test-1344.test.com |
| CPU: 1, Memory: 1 | test-1321.test.com |
| CPU: 1, Memory: 1 | test-1292.test.com |
| CPU: 1, Memory: 1 | test-1200.test.com |
| CPU: 1, Memory: 1 | test-1198.test.com |
| CPU: 1, Memory: 1 | test-1178.test.com |
| CPU: 1, Memory: 1 | test-1474.test.com |
| CPU: 1, Memory: 1 | test-1344.test.com |
| CPU: 1, Memory: 1 | test-1321.test.com |
+-------------------+--------------------------------------+
================================= Failed Pods =================================
Pods summary:
	- Unschedulable: 10
+----------------------+------+
|     REQUIREMENTS     | HOST |
+----------------------+------+
| CPU: 100, Memory: 1k |      |
| CPU: 100, Memory: 1k |      |
| CPU: 100, Memory: 1k |      |
| CPU: 100, Memory: 1k |      |
| CPU: 100, Memory: 1k |      |
| CPU: 100, Memory: 1k |      |
| CPU: 100, Memory: 1k |      |
| CPU: 100, Memory: 1k |      |
| CPU: 100, Memory: 1k |      |
| CPU: 100, Memory: 1k |      |
+----------------------+------+

```

# Notes
This project is inspired by [https://github.com/kubernetes-incubator/cluster-capacity](https://github.com/kubernetes-incubator/cluster-capacity).
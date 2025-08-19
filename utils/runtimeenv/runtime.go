package runtimeenv

import (
	"os"
	"strings"
)

const (
	Kubernetes = "kubernetes"
	Docker     = "docker"
	Source     = "source"
)

var runtimeEnv = runtimeEnvironment()

func isDocker() bool {
	data, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "docker")
}

func isKubernetes() bool {
	_, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount")
	return err == nil
}

func runtimeEnvironment() string {
	if isKubernetes() {
		return Kubernetes
	} else if isDocker() {
		return Docker
	} else {
		return Source
	}
}

func RuntimeEnvironment() string {
	return runtimeEnv
}

package runtimeenv

import (
	"os"
	"strings"
)

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

func PrintRuntimeEnvironment() string {
	var runtimeEnv string
	if isKubernetes() {
		// fmt.Println("Running inside Kubernetes")
		runtimeEnv = "kubernetes"
	} else if isDocker() {
		// fmt.Println("Running inside Docker")
		runtimeEnv = "docker"
	} else {
		// fmt.Println("Running in a local or non-Docker environment")
		runtimeEnv = "source"
	}

	return runtimeEnv
}

package manifestutils

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

// SetGoMemLimit sets GOMEMLIMIT env var to 80% memory of the container if it's defined.
func SetGoMemLimit(containerName string, pod *v1.PodSpec) {
	index, _ := findContainerIndex(pod, containerName)

	if index == -1 {
		return
	}

	container := &pod.Containers[index]

	memory := container.Resources.Limits.Memory()
	if memory != nil && !memory.IsZero() {
		bytes := memory.Value()
		gomemlimit := bytes * 80 / 100
		container.Env = append(container.Env, v1.EnvVar{
			Name:  "GOMEMLIMIT",
			Value: fmt.Sprintf("%d", gomemlimit),
		})
	}
}

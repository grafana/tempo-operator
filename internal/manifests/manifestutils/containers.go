package manifestutils

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// FindContainerIndex and return a container with the given name, or error if not found.
func FindContainerIndex(pod *corev1.PodSpec, containerName string) (int, error) {
	for i, container := range pod.Containers {
		if container.Name == containerName {
			return i, nil
		}
	}

	return -1, fmt.Errorf("cannot find container %s", containerName)
}

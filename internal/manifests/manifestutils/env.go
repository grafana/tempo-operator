package manifestutils

import (
	corev1 "k8s.io/api/core/v1"
)

// PatchEnvVars appends user-provided env vars to the named container and sets envFrom.
// This should be called after all operator-managed env vars have been set,
// so user env vars can override operator defaults if needed.
func PatchEnvVars(pod *corev1.PodSpec, containerName string, env []corev1.EnvVar, envFrom []corev1.EnvFromSource) {
	if len(env) == 0 && len(envFrom) == 0 {
		return
	}

	index, _ := findContainerIndex(pod, containerName)
	if index == -1 {
		return
	}

	container := &pod.Containers[index]
	if len(env) > 0 {
		container.Env = append(container.Env, env...)
	}
	if len(envFrom) > 0 {
		container.EnvFrom = append(container.EnvFrom, envFrom...)
	}
}

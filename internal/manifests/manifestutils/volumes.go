package manifestutils

import corev1 "k8s.io/api/core/v1"

// ContainsVolume return true if the volume with given name is present in the slice.
func ContainsVolume(volumes []corev1.Volume, volumeName string) bool {
	for _, volume := range volumes {
		if volume.Name == volumeName {
			return true
		}
	}

	return false
}

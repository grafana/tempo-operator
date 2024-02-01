package manifestutils

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

// ConfigureTLSVolumes mounts the CA ConfigMap and Certificate Secret in a pod.
func ConfigureTLSVolumes(
	pod *corev1.PodSpec,
	containerIdx int,
	tlsSpec v1alpha1.TLSSpec,
	tlsCADir string,
	tlsCertDir string,
	volumeNamePrefix string,
) {
	if tlsSpec.CA != "" {
		volumeName := fmt.Sprintf("%s-ca", volumeNamePrefix)
		pod.Containers[containerIdx].VolumeMounts = append(pod.Containers[containerIdx].VolumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: tlsCADir,
			ReadOnly:  true,
		})
		pod.Volumes = append(pod.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: tlsSpec.CA,
					},
				},
			},
		})
	}

	if tlsSpec.Cert != "" {
		volumeName := fmt.Sprintf("%s-cert", volumeNamePrefix)
		pod.Containers[containerIdx].VolumeMounts = append(pod.Containers[containerIdx].VolumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: tlsCertDir,
			ReadOnly:  true,
		})
		pod.Volumes = append(pod.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: tlsSpec.Cert,
				},
			},
		})
	}
}

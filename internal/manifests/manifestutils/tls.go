package manifestutils

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

// MountCAConfigMap mounts the CA ConfigMap in a pod.
func MountCAConfigMap(
	pod *corev1.PodSpec,
	containerName string,
	caConfigMap string,
	caDir string,
) error {
	containerIdx, err := findContainerIndex(pod, containerName)
	if err != nil {
		return err
	}

	volumeName := naming.DNSName(caConfigMap)

	pod.Containers[containerIdx].VolumeMounts = append(pod.Containers[containerIdx].VolumeMounts, corev1.VolumeMount{
		Name:      volumeName,
		MountPath: caDir,
		ReadOnly:  true,
	})

	if !containsVolume(pod, volumeName) {
		pod.Volumes = append(pod.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: caConfigMap,
					},
				},
			},
		})
	}

	return nil
}

// MountCertSecret mounts the Certificate Secret in a pod.
func MountCertSecret(
	pod *corev1.PodSpec,
	containerName string,
	certSecret string,
	certDir string,
) error {
	containerIdx, err := findContainerIndex(pod, containerName)
	if err != nil {
		return err
	}

	volumeName := naming.DNSName(certSecret)

	pod.Containers[containerIdx].VolumeMounts = append(pod.Containers[containerIdx].VolumeMounts, corev1.VolumeMount{
		Name:      volumeName,
		MountPath: certDir,
		ReadOnly:  true,
	})

	if !containsVolume(pod, volumeName) {
		pod.Volumes = append(pod.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: certSecret,
				},
			},
		})
	}

	return nil
}

// MountTLSSpecVolumes mounts the CA ConfigMap and Certificate Secret in a pod.
func MountTLSSpecVolumes(
	pod *corev1.PodSpec,
	containerName string,
	tlsSpec v1alpha1.TLSSpec,
	caDir string,
	certDir string,
) error {
	if tlsSpec.CA != "" {
		err := MountCAConfigMap(pod, containerName, tlsSpec.CA, caDir)
		if err != nil {
			return err
		}
	}

	if tlsSpec.Cert != "" {
		err := MountCertSecret(pod, containerName, tlsSpec.Cert, certDir)
		if err != nil {
			return err
		}
	}

	return nil
}

// NewConfigMapCABundle creates a new ConfigMap with an annotation that triggers the
// service-ca-operator to inject the cluster CA bundle in this ConfigMap (service-ca.crt key).
func NewConfigMapCABundle(namespace string, name string, labels labels.Set) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: map[string]string{"service.beta.openshift.io/inject-cabundle": "true"},
		},
	}
}

func findContainerIndex(pod *corev1.PodSpec, containerName string) (int, error) {
	for i, container := range pod.Containers {
		if container.Name == containerName {
			return i, nil
		}
	}

	return -1, fmt.Errorf("cannot find container %s", containerName)
}

func containsVolume(pod *corev1.PodSpec, volumeName string) bool {
	for _, volume := range pod.Volumes {
		if volume.Name == volumeName {
			return true
		}
	}

	return false
}

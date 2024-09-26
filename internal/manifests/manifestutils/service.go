package manifestutils

import (
	"github.com/ViaQ/logerr/v2/kverrors"
	"github.com/imdario/mergo"
	corev1 "k8s.io/api/core/v1"

	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

// ConfigureServiceCAByContainerName modify the PodSpec adding the volumes and volumeMounts to the specified containers.
func ConfigureServiceCAByContainerName(podSpec *corev1.PodSpec, caBundleName string, containers ...string) error {
	targetContainers := map[string]struct{}{}
	for _, name := range containers {
		targetContainers[name] = struct{}{}
	}
	ids := []int{}
	for id, c := range podSpec.Containers {
		if _, exists := targetContainers[c.Name]; exists {
			ids = append(ids, id)
		}
	}
	return ConfigureServiceCA(podSpec, caBundleName, ids...)
}

// ConfigureServiceCA modify the PodSpec adding the volumes and volumeMounts to the specified containers.
func ConfigureServiceCA(podSpec *corev1.PodSpec, caBundleName string, containers ...int) error {
	secretVolumeSpec := corev1.PodSpec{
		Volumes: []corev1.Volume{
			{
				Name: caBundleName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: caBundleName,
						},
					},
				},
			},
		},
	}

	secretContainerSpec := corev1.Container{
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      caBundleName,
				ReadOnly:  false,
				MountPath: TempoInternalTLSCADir,
			},
		},
	}

	if err := mergo.Merge(podSpec, secretVolumeSpec, mergo.WithAppendSlice); err != nil {
		return kverrors.Wrap(err, "failed to merge volumes")
	}

	containersSlice := []int{}
	containersSlice = append(containersSlice, containers...)
	nContainers := len(podSpec.Containers)

	if len(containersSlice) == 0 {
		containersSlice = append(containersSlice, 0)
	}

	for _, i := range containersSlice {
		if i >= nContainers {
			continue
		}
		if err := mergo.Merge(&podSpec.Containers[i], secretContainerSpec, mergo.WithAppendSlice); err != nil {
			return kverrors.Wrap(err, "failed to merge container")
		}
	}
	return nil
}

// ConfigureServicePKIByContainerName modify the PodSpec adding cert the volumes and volumeMounts to the specified containers.
func ConfigureServicePKIByContainerName(tempoStackName string, component string, podSpec *corev1.PodSpec, containers ...string) error {
	targetContainers := map[string]struct{}{}
	for _, name := range containers {
		targetContainers[name] = struct{}{}
	}
	ids := []int{}
	for id, c := range podSpec.Containers {
		if _, exists := targetContainers[c.Name]; exists {
			ids = append(ids, id)
		}
	}
	return ConfigureServicePKI(tempoStackName, component, podSpec, ids...)
}

// ConfigureServicePKI modify the PodSpec adding cert the volumes and volumeMounts to the specified containers.
func ConfigureServicePKI(tempoStackName string, component string, podSpec *corev1.PodSpec, containers ...int) error {
	serviceName := naming.TLSSecretName(component, tempoStackName)
	secretVolumeSpec := corev1.PodSpec{
		Volumes: []corev1.Volume{
			{
				Name: serviceName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: serviceName,
					},
				},
			},
		},
	}
	secretContainerSpec := corev1.Container{
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      serviceName,
				ReadOnly:  false,
				MountPath: TempoInternalTLSCertDir,
			},
		},
	}

	if err := mergo.Merge(podSpec, secretVolumeSpec, mergo.WithAppendSlice); err != nil {
		return kverrors.Wrap(err, "failed to merge volumes")
	}

	containersSlice := []int{}
	containersSlice = append(containersSlice, containers...)
	nContainers := len(podSpec.Containers)

	if len(containers) == 0 {
		containersSlice = append(containersSlice, 0)
	}
	for _, i := range containersSlice {
		if i >= nContainers {
			continue
		}
		if err := mergo.Merge(&podSpec.Containers[i], secretContainerSpec, mergo.WithAppendSlice); err != nil {
			return kverrors.Wrap(err, "failed to merge container")
		}
	}
	return nil
}

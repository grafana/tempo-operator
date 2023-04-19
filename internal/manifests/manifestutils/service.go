package manifestutils

import (
	"path"

	"github.com/ViaQ/logerr/v2/kverrors"
	"github.com/imdario/mergo"
	corev1 "k8s.io/api/core/v1"

	"github.com/os-observability/tempo-operator/internal/manifests/naming"
)

// TempoServerTLSDir returns the mount path of the HTTP service certificates.
func TempoServerTLSDir() string {
	return path.Join(TLSDir, "server")
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
				MountPath: CABundleDir,
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
				MountPath: TempoServerTLSDir(),
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

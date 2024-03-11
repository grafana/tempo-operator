package manifestutils

import (
	"path"

	corev1 "k8s.io/api/core/v1"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

// ConfigureAzureStorage mounts the Azure Storage credentials in a pod.
func ConfigureAzureStorage(pod *corev1.PodSpec, containerName string, storageSecretName string, tlsSpec *v1alpha1.TLSSpec) error {
	containerIdx, err := findContainerIndex(pod, containerName)
	if err != nil {
		return err
	}

	pod.Containers[containerIdx].Env = append(pod.Containers[containerIdx].Env, []corev1.EnvVar{
		{
			Name: "AZURE_ACCOUNT_NAME",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "account_name",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: storageSecretName,
					},
				},
			},
		},
		{
			Name: "AZURE_ACCOUNT_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "account_key",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: storageSecretName,
					},
				},
			},
		},
	}...)
	pod.Containers[containerIdx].Args = append(pod.Containers[containerIdx].Args, []string{
		"--storage.trace.azure.storage_account_name=$(AZURE_ACCOUNT_NAME)",
		"--storage.trace.azure.storage_account_key=$(AZURE_ACCOUNT_KEY)",
	}...)
	return nil
}

// ConfigureGCS mounts the Google Cloud Storage credentials in a pod.
func ConfigureGCS(pod *corev1.PodSpec, containerName string, storageSecretName string, tlsSpec *v1alpha1.TLSSpec) error {
	secretVolumeName := "storage-gcs-key"      // nolint #nosec
	secretDirectory := "/etc/storage/secrets/" // nolint #nosec
	secretFile := path.Join(secretDirectory, "key.json")

	containerIdx, err := findContainerIndex(pod, containerName)
	if err != nil {
		return err
	}

	pod.Containers[containerIdx].Env = append(pod.Containers[containerIdx].Env, []corev1.EnvVar{
		{
			Name:  "GOOGLE_APPLICATION_CREDENTIALS",
			Value: secretFile,
		},
	}...)

	pod.Containers[containerIdx].VolumeMounts = append(pod.Containers[containerIdx].VolumeMounts, corev1.VolumeMount{
		Name:      secretVolumeName,
		ReadOnly:  true,
		MountPath: secretDirectory,
	})
	pod.Volumes = append(pod.Volumes, corev1.Volume{
		Name: secretVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: storageSecretName,
			},
		},
	})
	return nil
}

// ConfigureS3Storage mounts the Amazon S3 credentials and TLS certs in a pod.
func ConfigureS3Storage(pod *corev1.PodSpec, containerName string, storageSecretName string, tlsSpec *v1alpha1.TLSSpec) error {
	containerIdx, err := findContainerIndex(pod, containerName)
	if err != nil {
		return err
	}

	pod.Containers[containerIdx].Env = append(pod.Containers[containerIdx].Env, []corev1.EnvVar{
		{
			Name: "S3_SECRET_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "access_key_secret",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: storageSecretName,
					},
				},
			},
		},
		{
			Name: "S3_ACCESS_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "access_key_id",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: storageSecretName,
					},
				},
			},
		},
	}...)
	pod.Containers[containerIdx].Args = append(pod.Containers[containerIdx].Args, []string{
		"--storage.trace.s3.secret_key=$(S3_SECRET_KEY)",
		"--storage.trace.s3.access_key=$(S3_ACCESS_KEY)",
	}...)

	if tlsSpec != nil && tlsSpec.Enabled {
		err := MountTLSSpecVolumes(pod, containerName, *tlsSpec, StorageTLSCADir, StorageTLSCertDir)
		if err != nil {
			return err
		}
	}
	return nil
}

// ConfigureStorage configures storage.
func ConfigureStorage(tempo v1alpha1.TempoStack, pod *corev1.PodSpec, containerName string) error {
	if tempo.Spec.Storage.Secret.Name != "" {
		var configure func(pod *corev1.PodSpec, containerName string, storageSecretName string, tlsSpec *v1alpha1.TLSSpec) error
		switch tempo.Spec.Storage.Secret.Type {
		case v1alpha1.ObjectStorageSecretAzure:
			configure = ConfigureAzureStorage
		case v1alpha1.ObjectStorageSecretGCS:
			configure = ConfigureGCS
		case v1alpha1.ObjectStorageSecretS3:
			configure = ConfigureS3Storage
		}

		return configure(pod, containerName, tempo.Spec.Storage.Secret.Name, &tempo.Spec.Storage.TLS)
	}
	return nil
}

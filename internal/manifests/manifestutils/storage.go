package manifestutils

import (
	"path"

	"github.com/ViaQ/logerr/v2/kverrors"
	"github.com/imdario/mergo"
	corev1 "k8s.io/api/core/v1"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

const (
	storageCAVolumeName = "storage-ca"
)

// TempoStorageTLSDir returns the mount path of certificates for connecting to object storage.
func TempoStorageTLSDir() string {
	return path.Join(TLSDir, "storage")
}

// TempoStorageTLSCAPath returns the path of the CA certificate for connecting to object storage.
func TempoStorageTLSCAPath() string {
	return path.Join(TempoStorageTLSDir(), "ca.crt")
}

func configureAzureStorage(tempo *v1alpha1.TempoStack, pod *corev1.PodSpec) error {
	var envVars []corev1.EnvVar = []corev1.EnvVar{
		{
			Name: "AZURE_ACCOUNT_NAME",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "account_name",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: tempo.Spec.Storage.Secret.Name,
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
						Name: tempo.Spec.Storage.Secret.Name,
					},
				},
			},
		},
	}
	args := []string{
		"--storage.trace.azure.storage_account_name=$(AZURE_ACCOUNT_NAME)",
		"--storage.trace.azure.storage_account_key=$(AZURE_ACCOUNT_KEY)",
	}

	ingesterContainer := pod.Containers[0].DeepCopy()
	ingesterContainer.Env = append(ingesterContainer.Env, envVars...)
	ingesterContainer.Args = append(ingesterContainer.Args, args...)

	if err := mergo.Merge(&pod.Containers[0], ingesterContainer, mergo.WithOverride); err != nil {
		return kverrors.Wrap(err, "failed to merge ingester container spec")
	}
	return nil
}

func configureGCS(tempo *v1alpha1.TempoStack, pod *corev1.PodSpec) error {
	secretDirectory := "/etc/storage/secrets/" // nolint #nosec
	secretFile := path.Join(secretDirectory, "key.json")

	var envVars []corev1.EnvVar = []corev1.EnvVar{
		{
			Name:  "GOOGLE_APPLICATION_CREDENTIALS",
			Value: secretFile,
		},
	}

	var volumeMounts []corev1.VolumeMount = []corev1.VolumeMount{
		{
			Name:      tempo.Spec.Storage.Secret.Name,
			ReadOnly:  false,
			MountPath: secretDirectory,
		},
	}

	var volumes []corev1.Volume = []corev1.Volume{
		{
			Name: tempo.Spec.Storage.Secret.Name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: tempo.Spec.Storage.Secret.Name,
				},
			},
		},
	}

	ingesterContainer := pod.Containers[0].DeepCopy()
	ingesterContainer.Env = append(ingesterContainer.Env, envVars...)
	ingesterContainer.VolumeMounts = append(ingesterContainer.VolumeMounts, volumeMounts...)

	pod.Volumes = append(pod.Volumes, volumes...)

	if err := mergo.Merge(&pod.Containers[0], ingesterContainer, mergo.WithOverride); err != nil {
		return kverrors.Wrap(err, "failed to merge ingester container spec")
	}
	return nil
}

func configureS3Storage(tempo *v1alpha1.TempoStack, pod *corev1.PodSpec) error {
	var envVars []corev1.EnvVar = []corev1.EnvVar{
		{
			Name: "S3_SECRET_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "access_key_secret",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: tempo.Spec.Storage.Secret.Name,
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
						Name: tempo.Spec.Storage.Secret.Name,
					},
				},
			},
		},
	}
	args := []string{
		"--storage.trace.s3.secret_key=$(S3_SECRET_KEY)",
		"--storage.trace.s3.access_key=$(S3_ACCESS_KEY)",
	}

	volumeMounts := []corev1.VolumeMount{}
	volumes := []corev1.Volume{}
	if tempo.Spec.Storage.TLS.CA != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      storageCAVolumeName,
			MountPath: TempoStorageTLSDir(),
			ReadOnly:  true,
		})
		volumes = append(volumes, corev1.Volume{
			Name: storageCAVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: tempo.Spec.Storage.TLS.CA,
					},
				},
			},
		})
	}

	ingesterContainer := pod.Containers[0].DeepCopy()
	ingesterContainer.Env = append(ingesterContainer.Env, envVars...)
	ingesterContainer.Args = append(ingesterContainer.Args, args...)
	ingesterContainer.VolumeMounts = append(ingesterContainer.VolumeMounts, volumeMounts...)
	pod.Volumes = append(pod.Volumes, volumes...)

	if err := mergo.Merge(&pod.Containers[0], ingesterContainer, mergo.WithOverride); err != nil {
		return kverrors.Wrap(err, "failed to merge ingester container spec")
	}
	return nil
}

// ConfigureStorage configures storage.
func ConfigureStorage(tempo v1alpha1.TempoStack, pod *corev1.PodSpec) error {
	if tempo.Spec.Storage.Secret.Name != "" {
		var configure func(*v1alpha1.TempoStack, *corev1.PodSpec) error
		switch tempo.Spec.Storage.Secret.Type {
		case v1alpha1.ObjectStorageSecretAzure:
			configure = configureAzureStorage
		case v1alpha1.ObjectStorageSecretGCS:
			configure = configureGCS
		case v1alpha1.ObjectStorageSecretS3:
			configure = configureS3Storage
		}

		return configure(&tempo, pod)
	}
	return nil
}

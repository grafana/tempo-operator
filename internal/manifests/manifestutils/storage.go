package manifestutils

import (
	"fmt"
	"path"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func configureAzureStaticTokenStorage(pod *corev1.PodSpec, containerName string, storageSecretName string) error {
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

func configureAzureShortTokenStorage(pod *corev1.PodSpec, params *AzureStorage, containerName string, storageSecretName string) error {
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
			Name: "AZURE_CLIENT_ID",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "client_id",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: storageSecretName,
					},
				},
			},
		},
		{
			Name: "AZURE_TENANT_ID",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "tenant_id",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: storageSecretName,
					},
				},
			},
		},
		{
			Name:  "AZURE_FEDERATED_TOKEN_FILE",
			Value: ServiceAccountTokenFilePath,
		},
	}...)
	pod.Containers[containerIdx].Args = append(pod.Containers[containerIdx].Args, []string{
		"--storage.trace.azure.storage_account_name=$(AZURE_ACCOUNT_NAME)",
	}...)

	pod.Volumes = append(pod.Volumes, saTokenVolume(params.Audience))

	pod.Containers[containerIdx].VolumeMounts = append(pod.Containers[containerIdx].VolumeMounts, corev1.VolumeMount{
		Name:      saTokenVolumeName,
		MountPath: saTokenVolumeMountPath,
	})

	return nil
}

// ConfigureAzureStorage mounts the Azure Storage credentials in a pod.
func ConfigureAzureStorage(podTemplate *corev1.PodSpec, params *AzureStorage, containerName string, storageSecretName string,
	credentialMode v1alpha1.CredentialMode) error {
	if credentialMode == v1alpha1.CredentialModeToken {
		return configureAzureShortTokenStorage(podTemplate, params, containerName, storageSecretName)
	}

	return configureAzureStaticTokenStorage(podTemplate, containerName, storageSecretName)
}

// ConfigureGCS mounts the Google Cloud Storage credentials in a pod.
func ConfigureGCS(pod *corev1.PodSpec, containerName string, storageSecretName string, audience string, credentialMode v1alpha1.CredentialMode) error {
	secretVolumeName := "storage-gcs-key"      // nolint #nosec
	secretDirectory := "/etc/storage/secrets/" // nolint #nosec
	secretFile := path.Join(secretDirectory, "key.json")

	containerIdx, err := findContainerIndex(pod, containerName)

	if err != nil {
		return err
	}

	if credentialMode == v1alpha1.CredentialModeStatic || credentialMode == v1alpha1.CredentialModeToken {

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
	}

	if credentialMode == v1alpha1.CredentialModeToken {
		pod.Volumes = append(pod.Volumes, saTokenVolume(audience))

		pod.Containers[containerIdx].VolumeMounts = append(pod.Containers[containerIdx].VolumeMounts, corev1.VolumeMount{
			Name:      saTokenVolumeName,
			MountPath: saTokenVolumeMountPath,
		})

	}
	return nil
}

func configureS3StorageWithCOOAuth(pod *corev1.PodSpec, containerIdx int, tempo string, config *TokenCCOAuthConfig) {

	pod.Containers[containerIdx].Env = append(pod.Containers[containerIdx].Env, []corev1.EnvVar{
		{
			Name:  "AWS_SHARED_CREDENTIALS_FILE",
			Value: path.Join(tokenAuthConfigDirectory, "credentials"),
		},
		{
			Name:  "AWS_SDK_LOAD_CONFIG",
			Value: "true",
		},
		{
			Name:  "AWS_WEB_IDENTITY_TOKEN_FILE",
			Value: ServiceAccountTokenFilePath,
		},
		{
			Name:  "AWS_ROLE_ARN",
			Value: config.AWS.RoleARN,
		},
	}...)

	// Define volume with credentials
	pod.Volumes = append(pod.Volumes, tokenCCOAuthConfigVolume(tempo))
	pod.Volumes = append(pod.Volumes, saTokenVolume(awsDefaultAudience))

	// Mount volume
	pod.Containers[containerIdx].VolumeMounts = append(pod.Containers[containerIdx].VolumeMounts, corev1.VolumeMount{
		Name:      tokenAuthConfigVolumeName,
		MountPath: tokenAuthConfigDirectory,
	})

	pod.Containers[containerIdx].VolumeMounts = append(pod.Containers[containerIdx].VolumeMounts, corev1.VolumeMount{
		Name:      saTokenVolumeName,
		MountPath: saTokenVolumeMountPath,
	})
}

func configureS3StorageStatic(pod *corev1.PodSpec, containerIdx int, storageSecretName string) {
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
}

// ConfigureS3Storage mounts the Amazon S3 credentials and TLS certs in a pod.
func ConfigureS3Storage(pod *corev1.PodSpec, containerName string, storageSecretName string,
	tlsSpec *v1alpha1.TLSSpec, credentialMode v1alpha1.CredentialMode, tempoName string, config *TokenCCOAuthConfig) error {

	if credentialMode == v1alpha1.CredentialModeToken {
		return nil
	}

	containerIdx, err := findContainerIndex(pod, containerName)
	if err != nil {
		return err
	}

	if credentialMode == v1alpha1.CredentialModeTokenCCO {
		configureS3StorageWithCOOAuth(pod, containerIdx, tempoName, config)
	} else {
		configureS3StorageStatic(pod, containerIdx, storageSecretName)
	}

	if tlsSpec != nil && tlsSpec.Enabled {
		err := MountTLSSpecVolumes(pod, containerName, *tlsSpec, StorageTLSCADir, StorageTLSCertDir)
		if err != nil {
			return err
		}
	}
	return nil
}

// ConfigureStorage configures storage.
func ConfigureStorage(storage StorageParams, tempo v1alpha1.TempoStack, pod *corev1.PodSpec, containerName string) error {
	if tempo.Spec.Storage.Secret.Name != "" {
		switch tempo.Spec.Storage.Secret.Type {
		case v1alpha1.ObjectStorageSecretAzure:
			return ConfigureAzureStorage(pod, storage.AzureStorage, containerName, tempo.Spec.Storage.Secret.Name, storage.CredentialMode)
		case v1alpha1.ObjectStorageSecretGCS:
			return ConfigureGCS(pod, containerName, tempo.Spec.Storage.Secret.Name, storage.GCS.Audience,
				storage.CredentialMode)
		case v1alpha1.ObjectStorageSecretS3:
			return ConfigureS3Storage(pod, containerName, tempo.Spec.Storage.Secret.Name, &tempo.Spec.Storage.TLS,
				storage.CredentialMode, tempo.Name, storage.CloudCredentials.Environment)
		}
	}
	return nil
}

func tokenCCOAuthConfigVolume(tempo string) corev1.Volume {
	return corev1.Volume{
		Name: tokenAuthConfigVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: ManagedCredentialsSecretName(tempo),
			},
		},
	}
}

// ManagedCredentialsSecretName secret name with credentials.
func ManagedCredentialsSecretName(stackName string) string {
	return fmt.Sprintf("%s-managed-credentials", stackName)
}

// TempoFromManagerCredentialSecretName tempo stack from secret name.
func TempoFromManagerCredentialSecretName(secretName string) string {
	return strings.TrimSuffix(secretName, "-managed-credentials")
}

func saTokenVolume(audience string) corev1.Volume {
	return corev1.Volume{
		Name: saTokenVolumeName,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
							ExpirationSeconds: ptr.To(saTokenExpiration),
							Path:              corev1.ServiceAccountTokenKey,
							Audience:          audience,
						},
					},
				},
			},
		},
	}
}

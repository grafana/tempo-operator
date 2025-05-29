package manifestutils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func findEnvVar(name string, envVars *[]corev1.EnvVar) error {
	for _, envVar := range *envVars {
		if envVar.Name == name {
			return nil
		}
	}
	return fmt.Errorf("%s environment variable not found in list", name)
}

func TestConfigureAzureStorage(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: "test",
					Type: v1alpha1.ObjectStorageSecretAzure,
				},
			},
		},
	}

	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name: "ingester",
			},
		},
	}

	assert.NoError(t, ConfigureAzureStorage(&pod, &AzureStorage{}, "ingester", tempo.Spec.Storage.Secret.Name, v1alpha1.CredentialModeStatic))
	assert.Len(t, pod.Containers[0].Env, 2)
	assert.NoError(t, findEnvVar("AZURE_ACCOUNT_NAME", &pod.Containers[0].Env))
	assert.NoError(t, findEnvVar("AZURE_ACCOUNT_KEY", &pod.Containers[0].Env))

	assert.Len(t, pod.Containers[0].Args, 2)
	assert.Contains(t, pod.Containers[0].Args, "--storage.trace.azure.storage_account_name=$(AZURE_ACCOUNT_NAME)")
	assert.Contains(t, pod.Containers[0].Args, "--storage.trace.azure.storage_account_key=$(AZURE_ACCOUNT_KEY)")
}

func TestGetGCSStorage(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: "test",
					Type: v1alpha1.ObjectStorageSecretGCS,
				},
			},
		},
	}

	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name: "ingester",
			},
		},
	}

	assert.NoError(t, ConfigureGCS(&pod, "ingester", tempo.Spec.Storage.Secret.Name, "audience",
		v1alpha1.CredentialModeStatic))
	assert.Len(t, pod.Containers[0].Env, 1)
	assert.NoError(t, findEnvVar("GOOGLE_APPLICATION_CREDENTIALS", &pod.Containers[0].Env))

	assert.Len(t, pod.Containers[0].VolumeMounts, 1)
}

func TestGetS3Storage(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					CredentialMode: v1alpha1.CredentialModeStatic,
					Name:           "test",
					Type:           v1alpha1.ObjectStorageSecretAzure,
				},
			},
		},
	}

	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name: "ingester",
			},
		},
	}

	assert.NoError(t, ConfigureS3Storage(&pod, "ingester", tempo.Spec.Storage.Secret.Name, &tempo.Spec.Storage.TLS, tempo.Spec.Storage.Secret.CredentialMode, "", &TokenCCOAuthConfig{}))
	assert.Len(t, pod.Containers[0].Env, 2)
	assert.NoError(t, findEnvVar("S3_SECRET_KEY", &pod.Containers[0].Env))
	assert.NoError(t, findEnvVar("S3_ACCESS_KEY", &pod.Containers[0].Env))

	assert.Len(t, pod.Containers[0].Args, 2)
	assert.Contains(t, pod.Containers[0].Args, "--storage.trace.s3.secret_key=$(S3_SECRET_KEY)")
	assert.Contains(t, pod.Containers[0].Args, "--storage.trace.s3.access_key=$(S3_ACCESS_KEY)")
}

func TestGetS3Storage_short_lived(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					CredentialMode: v1alpha1.CredentialModeToken,
					Name:           "test",
					Type:           v1alpha1.ObjectStorageSecretAzure,
				},
			},
		},
	}

	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name: "ingester",
			},
		},
	}

	assert.NoError(t, ConfigureS3Storage(&pod, "ingester", tempo.Spec.Storage.Secret.Name, &tempo.Spec.Storage.TLS, tempo.Spec.Storage.Secret.CredentialMode, "", &TokenCCOAuthConfig{}))
	assert.Len(t, pod.Containers[0].Env, 0)
	assert.Len(t, pod.Containers[0].Args, 0)
}

func TestGetS3StorageWithCA(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: "test",
					Type: v1alpha1.ObjectStorageSecretAzure,
				},
				TLS: v1alpha1.TLSSpec{
					Enabled: true,
					CA:      "customca",
				},
			},
		},
	}

	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name: "ingester",
			},
		},
	}

	assert.NoError(t, ConfigureS3Storage(&pod, "ingester", tempo.Spec.Storage.Secret.Name, &tempo.Spec.Storage.TLS, tempo.Spec.Storage.Secret.CredentialMode, "", &TokenCCOAuthConfig{}))
	assert.Equal(t, []corev1.Volume{
		{
			Name: "customca",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: tempo.Spec.Storage.TLS.CA,
					},
				},
			},
		},
	}, pod.Volumes)

	assert.Len(t, pod.Containers[0].VolumeMounts, 1)
	assert.Equal(t, []corev1.VolumeMount{
		{
			Name:      "customca",
			MountPath: StorageTLSCADir,
			ReadOnly:  true,
		},
	}, pod.Containers[0].VolumeMounts)
}

func TestConfigureStorage(t *testing.T) {
	tests := []struct {
		name    string
		tempo   v1alpha1.TempoStack
		pod     corev1.PodSpec
		envName string
	}{
		{
			name: "Azure Storage configuration",
			tempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Storage: v1alpha1.ObjectStorageSpec{
						Secret: v1alpha1.ObjectStorageSecretSpec{
							Name:           "test",
							Type:           v1alpha1.ObjectStorageSecretAzure,
							CredentialMode: v1alpha1.CredentialModeStatic,
						},
					},
				},
			},
			pod: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "ingester",
					},
				},
			},
			envName: "AZURE_ACCOUNT_NAME",
		},
		{
			name: "Google Cloud Storage configuration",
			tempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Storage: v1alpha1.ObjectStorageSpec{
						Secret: v1alpha1.ObjectStorageSecretSpec{
							Name:           "test",
							Type:           v1alpha1.ObjectStorageSecretGCS,
							CredentialMode: v1alpha1.CredentialModeStatic,
						},
					},
				},
			},
			pod: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "ingester",
					},
				},
			},
			envName: "GOOGLE_APPLICATION_CREDENTIALS",
		},
		{
			name: "S3 Storage configuration",
			tempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Storage: v1alpha1.ObjectStorageSpec{
						Secret: v1alpha1.ObjectStorageSecretSpec{
							Name:           "test",
							Type:           v1alpha1.ObjectStorageSecretS3,
							CredentialMode: v1alpha1.CredentialModeStatic,
						},
					},
				},
			},
			pod: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "ingester",
					},
				},
			},
			envName: "S3_SECRET_KEY",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NoError(t, ConfigureStorage(StorageParams{
				CredentialMode: test.tempo.Spec.Storage.Secret.CredentialMode,
				GCS: &GCS{
					Audience: "openshift",
				},
			}, test.tempo, &test.pod, "ingester"))
			assert.NoError(t, findEnvVar(test.envName, &test.pod.Containers[0].Env))
		})
	}

}

func TestGetCGSStorage_short_lived(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name:           "test",
					Type:           v1alpha1.ObjectStorageSecretGCS,
					CredentialMode: v1alpha1.CredentialModeToken,
				},
			},
		},
	}

	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name: "ingester",
			},
		},
	}

	assert.NoError(t, ConfigureGCS(&pod, "ingester", tempo.Spec.Storage.Secret.Name, "audiente",
		v1alpha1.CredentialModeToken))
	assert.Len(t, pod.Containers[0].Env, 1)
	assert.Len(t, pod.Containers[0].Args, 0)
}

func TestConfigureStorageWithS3CCO(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					CredentialMode: v1alpha1.CredentialModeTokenCCO,
					Name:           "test",
					Type:           v1alpha1.ObjectStorageSecretS3,
				},
			},
		},
	}

	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name: "ingester",
			},
		},
	}

	assert.NoError(t, ConfigureS3Storage(&pod, "ingester",
		tempo.Spec.Storage.Secret.Name, &tempo.Spec.Storage.TLS, tempo.Spec.Storage.Secret.CredentialMode, "", &TokenCCOAuthConfig{
			AWS: &TokenCCOAWSEnvironment{
				RoleARN: "arn:aws:iam::12345:role/test",
			},
		}))
	assert.Len(t, pod.Containers[0].Env, 4)
	assert.NoError(t, findEnvVar("AWS_SHARED_CREDENTIALS_FILE", &pod.Containers[0].Env))
	assert.NoError(t, findEnvVar("AWS_SDK_LOAD_CONFIG", &pod.Containers[0].Env))
	assert.Len(t, pod.Containers[0].VolumeMounts, 2)
	assert.Len(t, pod.Volumes, 2)
	assert.Equal(t, tokenAuthConfigVolumeName, pod.Containers[0].VolumeMounts[0].Name)
	assert.Equal(t, tokenAuthConfigVolumeName, pod.Volumes[0].Name)
	assert.NotContains(t, pod.Containers[0].Args, "--storage.trace.s3.secret_key=$(S3_SECRET_KEY)")
	assert.NotContains(t, pod.Containers[0].Args, "--storage.trace.s3.access_key=$(S3_ACCESS_KEY)")
}

func TestAzureStorage_short_lived(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name:           "test",
					Type:           v1alpha1.ObjectStorageSecretAzure,
					CredentialMode: v1alpha1.CredentialModeToken,
				},
			},
		},
	}

	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name: "ingester",
			},
		},
	}

	params := StorageParams{
		CredentialMode: tempo.Spec.Storage.Secret.CredentialMode,
		AzureStorage: &AzureStorage{
			TenantID:  "test-tenant",
			ClientID:  "test-client",
			Container: "tempo",
		},
	}
	assert.NoError(t, ConfigureStorage(params, tempo, &pod, "ingester"))
	assert.Equal(t, pod.Containers[0].Env, []corev1.EnvVar{
		{
			Name: "AZURE_ACCOUNT_NAME",
			ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
				Key: "account_name",
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "test",
				},
			}},
		},
		{
			Name: "AZURE_CLIENT_ID",
			ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
				Key: "client_id",
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "test",
				},
			}},
		},
		{
			Name: "AZURE_TENANT_ID",
			ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
				Key: "tenant_id",
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "test",
				},
			}},
		},
		{
			Name:  "AZURE_FEDERATED_TOKEN_FILE",
			Value: "/var/run/secrets/storage/serviceaccount/token",
		},
	})
	assert.Len(t, pod.Containers[0].VolumeMounts, 1)
	assert.Len(t, pod.Volumes, 1)
	assert.Equal(t, saTokenVolumeName, pod.Containers[0].VolumeMounts[0].Name)
	assert.Equal(t, saTokenVolumeName, pod.Volumes[0].Name)
	assert.Contains(t, pod.Containers[0].Args, "--storage.trace.azure.storage_account_name=$(AZURE_ACCOUNT_NAME)")

}

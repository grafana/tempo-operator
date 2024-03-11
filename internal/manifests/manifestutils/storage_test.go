package manifestutils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
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

	assert.NoError(t, ConfigureAzureStorage(&pod, "ingester", tempo.Spec.Storage.Secret.Name, &tempo.Spec.Storage.TLS))
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

	assert.NoError(t, ConfigureGCS(&pod, "ingester", tempo.Spec.Storage.Secret.Name, &tempo.Spec.Storage.TLS))
	assert.Len(t, pod.Containers[0].Env, 1)
	assert.NoError(t, findEnvVar("GOOGLE_APPLICATION_CREDENTIALS", &pod.Containers[0].Env))

	assert.Len(t, pod.Containers[0].VolumeMounts, 1)
}

func TestGetS3Storage(t *testing.T) {
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

	assert.NoError(t, ConfigureS3Storage(&pod, "ingester", tempo.Spec.Storage.Secret.Name, &tempo.Spec.Storage.TLS))
	assert.Len(t, pod.Containers[0].Env, 2)
	assert.NoError(t, findEnvVar("S3_SECRET_KEY", &pod.Containers[0].Env))
	assert.NoError(t, findEnvVar("S3_ACCESS_KEY", &pod.Containers[0].Env))

	assert.Len(t, pod.Containers[0].Args, 2)
	assert.Contains(t, pod.Containers[0].Args, "--storage.trace.s3.secret_key=$(S3_SECRET_KEY)")
	assert.Contains(t, pod.Containers[0].Args, "--storage.trace.s3.access_key=$(S3_ACCESS_KEY)")
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

	assert.NoError(t, ConfigureS3Storage(&pod, "ingester", tempo.Spec.Storage.Secret.Name, &tempo.Spec.Storage.TLS))
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
							Name: "test",
							Type: v1alpha1.ObjectStorageSecretAzure,
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
							Name: "test",
							Type: v1alpha1.ObjectStorageSecretGCS,
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
							Name: "test",
							Type: v1alpha1.ObjectStorageSecretS3,
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
			assert.NoError(t, ConfigureStorage(test.tempo, &test.pod, "ingester"))
			assert.NoError(t, findEnvVar(test.envName, &test.pod.Containers[0].Env))
		})
	}

}

package manifestutils

import (
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func findEnvVar(name string, envVars *[]corev1.EnvVar) error {
	for _, envVar := range *envVars {
		if envVar.Name == name {
			return nil
		}
	}
	return fmt.Errorf("%s environment variable not found in list", name)
}

func TestGetS3Storage(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: "test",
					Type: v1alpha1.ObjectStorageSecretS3,
				},
			},
		},
	}

	envVars, args := getS3Storage(&tempo)

	assert.Len(t, envVars, 2)
	assert.NoError(t, findEnvVar("S3_SECRET_KEY", &envVars))
	assert.NoError(t, findEnvVar("S3_ACCESS_KEY", &envVars))

	assert.Len(t, args, 2)
	assert.Contains(t, args, "--storage.trace.s3.secret_key=$(S3_SECRET_KEY)")
	assert.Contains(t, args, "--storage.trace.s3.access_key=$(S3_ACCESS_KEY)")
}

func TestConfigureStorage(t *testing.T) {
	tests := []struct {
		name  string
		tempo v1alpha1.TempoStack
		pod   corev1.PodSpec
	}{
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
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NoError(t, ConfigureStorage(test.tempo, &test.pod))
			assert.Len(t, test.pod.Containers[0].Args, 2)
			assert.Len(t, test.pod.Containers[0].Env, 2)
		})
	}

}

package v1alpha1

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestDefault(t *testing.T) {
	defaulter := &defaulter{
		defaultImages: ImagesSpec{
			Tempo:      "docker.io/grafana/tempo:x.y.z",
			TempoQuery: "docker.io/grafana/tempo-query:x.y.z",
		},
	}

	defaultMaxSearch := 0
	tests := []struct {
		input    *Microservices
		expected *Microservices
		name     string
	}{
		{
			name: "no action default values are provided",
			input: &Microservices{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: MicroservicesSpec{
					Images: ImagesSpec{
						Tempo:      "docker.io/grafana/tempo:1.2.3",
						TempoQuery: "docker.io/grafana/tempo-query:1.2.3",
					},
					ServiceAccount: "tempo-test-serviceaccount",
					Retention: RetentionSpec{
						Global: RetentionConfig{
							Traces: time.Hour,
						},
					},
					StorageSize: resource.MustParse("1Gi"),
					LimitSpec: LimitSpec{
						Global: RateLimitSpec{
							Query: QueryLimit{
								MaxSearchBytesPerTrace: &defaultMaxSearch,
							},
						},
					},
				},
			},
			expected: &Microservices{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: MicroservicesSpec{
					Images: ImagesSpec{
						Tempo:      "docker.io/grafana/tempo:1.2.3",
						TempoQuery: "docker.io/grafana/tempo-query:1.2.3",
					},
					ServiceAccount: "tempo-test-serviceaccount",
					Retention: RetentionSpec{
						Global: RetentionConfig{
							Traces: time.Hour,
						},
					},
					StorageSize: resource.MustParse("1Gi"),
					LimitSpec: LimitSpec{
						Global: RateLimitSpec{
							Query: QueryLimit{
								MaxSearchBytesPerTrace: &defaultMaxSearch,
							},
						},
					},
				},
			},
		},
		{
			name: "default values are set in the webhook",
			input: &Microservices{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			expected: &Microservices{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: MicroservicesSpec{
					Images: ImagesSpec{
						Tempo:      "docker.io/grafana/tempo:x.y.z",
						TempoQuery: "docker.io/grafana/tempo-query:x.y.z",
					},
					ServiceAccount: "tempo-test-serviceaccount",
					Retention: RetentionSpec{
						Global: RetentionConfig{
							Traces: 48 * time.Hour,
						},
					},
					StorageSize: resource.MustParse("10Gi"),
					LimitSpec: LimitSpec{
						Global: RateLimitSpec{
							Query: QueryLimit{
								MaxSearchBytesPerTrace: &defaultMaxSearch,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := defaulter.Default(context.Background(), test.input)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, test.input)
		})
	}
}

func TestValidateStorageSecret(t *testing.T) {
	tempo := &Microservices{
		Spec: MicroservicesSpec{
			Storage: ObjectStorageSpec{
				Secret: "testsecret",
			},
		},
	}
	validator := &validator{}
	path := field.NewPath("spec").Child("storage").Child("secret")

	tests := []struct {
		name     string
		input    *corev1.Secret
		expected field.ErrorList
	}{
		{
			name:  "empty secret",
			input: &corev1.Secret{},
			expected: field.ErrorList{
				field.Invalid(path, tempo.Spec.Storage.Secret, "storage secret is empty"),
			},
		},
		{
			name: "missing or empty fields",
			input: &corev1.Secret{
				Data: map[string][]byte{
					"bucket": []byte(""),
				},
			},
			expected: field.ErrorList{
				field.Invalid(path, tempo.Spec.Storage.Secret, "storage secret must contain \"endpoint\" field"),
				field.Invalid(path, tempo.Spec.Storage.Secret, "storage secret must contain \"bucket\" field"),
				field.Invalid(path, tempo.Spec.Storage.Secret, "storage secret must contain \"access_key_id\" field"),
				field.Invalid(path, tempo.Spec.Storage.Secret, "storage secret must contain \"access_key_secret\" field"),
			},
		},
		{
			name: "invalid endpoint 'invalid'",
			input: &corev1.Secret{
				Data: map[string][]byte{
					"endpoint":          []byte("invalid"),
					"bucket":            []byte("bucket"),
					"access_key_id":     []byte("id"),
					"access_key_secret": []byte("secret"),
				},
			},
			expected: field.ErrorList{
				field.Invalid(path, tempo.Spec.Storage.Secret, "\"endpoint\" field of storage secret must be a valid URL"),
			},
		},
		{
			name: "invalid endpoint '/invalid'",
			input: &corev1.Secret{
				Data: map[string][]byte{
					"endpoint":          []byte("/invalid"),
					"bucket":            []byte("bucket"),
					"access_key_id":     []byte("id"),
					"access_key_secret": []byte("secret"),
				},
			},
			expected: field.ErrorList{
				field.Invalid(path, tempo.Spec.Storage.Secret, "\"endpoint\" field of storage secret must be a valid URL"),
			},
		},
		{
			name: "valid storage secret",
			input: &corev1.Secret{
				Data: map[string][]byte{
					"endpoint":          []byte("http://minio.minio.svc:9000"),
					"bucket":            []byte("bucket"),
					"access_key_id":     []byte("id"),
					"access_key_secret": []byte("secret"),
				},
			},
			expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := validator.validateStorageSecret(tempo, test.input)
			assert.Equal(t, test.expected, errs)
		})
	}
}

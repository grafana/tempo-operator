package v1alpha1

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

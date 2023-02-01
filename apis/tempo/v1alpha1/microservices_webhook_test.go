package v1alpha1

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
)

func TestDefault(t *testing.T) {
	defaulter := &Defaulter{
		defaultImages: ImagesSpec{
			Tempo:      "docker.io/grafana/tempo:x.y.z",
			TempoQuery: "docker.io/grafana/tempo-query:x.y.z",
		},
	}

	defaultSearchEnabled := true
	defaultMaxSearch := 0
	defaultDefaultResultLimit := 20

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
					ReplicationFactor: 2,
					Images: ImagesSpec{
						Tempo:      "docker.io/grafana/tempo:1.2.3",
						TempoQuery: "docker.io/grafana/tempo-query:1.2.3",
					},
					ServiceAccount: "tempo-test-serviceaccount",
					Retention: RetentionSpec{
						Global: RetentionConfig{
							Traces: metav1.Duration{Duration: time.Hour},
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
					ReplicationFactor: 2,
					Images: ImagesSpec{
						Tempo:      "docker.io/grafana/tempo:1.2.3",
						TempoQuery: "docker.io/grafana/tempo-query:1.2.3",
					},
					ServiceAccount: "tempo-test-serviceaccount",
					Retention: RetentionSpec{
						Global: RetentionConfig{
							Traces: metav1.Duration{Duration: time.Hour},
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
					SearchSpec: SearchSpec{
						MaxDuration:        metav1.Duration{Duration: 0},
						Enabled:            &defaultSearchEnabled,
						DefaultResultLimit: &defaultDefaultResultLimit,
					},
					Components: TempoComponentsSpec{
						Distributor: TempoComponentSpec{
							Replicas: pointer.Int32(1),
						},
						Ingester: TempoComponentSpec{
							Replicas: pointer.Int32(1),
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
					ReplicationFactor: 1,
					Images: ImagesSpec{
						Tempo:      "docker.io/grafana/tempo:x.y.z",
						TempoQuery: "docker.io/grafana/tempo-query:x.y.z",
					},
					ServiceAccount: "tempo-test-serviceaccount",
					Retention: RetentionSpec{
						Global: RetentionConfig{
							Traces: metav1.Duration{Duration: 48 * time.Hour},
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
					SearchSpec: SearchSpec{
						MaxDuration:        metav1.Duration{Duration: 0},
						Enabled:            &defaultSearchEnabled,
						DefaultResultLimit: &defaultDefaultResultLimit,
					},
					Components: TempoComponentsSpec{
						Distributor: TempoComponentSpec{
							Replicas: pointer.Int32(1),
						},
						Ingester: TempoComponentSpec{
							Replicas: pointer.Int32(1),
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

func TestValidateReplicationFactor(t *testing.T) {
	validator := &validator{}
	path := field.NewPath("spec").Child("ReplicationFactor")

	tests := []struct {
		name     string
		expected field.ErrorList
		input    Microservices
	}{
		{
			name: "no error replicas equal to floor(replication_factor/2) + 1",
			input: Microservices{
				Spec: MicroservicesSpec{
					ReplicationFactor: 3,
					Components: TempoComponentsSpec{
						Ingester: TempoComponentSpec{
							Replicas: pointer.Int32(2),
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "no error replicas greater than floor(replication_factor/2) + 1",
			input: Microservices{
				Spec: MicroservicesSpec{
					ReplicationFactor: 3,
					Components: TempoComponentsSpec{
						Ingester: TempoComponentSpec{
							Replicas: pointer.Int32(3),
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "error replicas less than floor(replication_factor/2) + 1",
			input: Microservices{
				Spec: MicroservicesSpec{
					ReplicationFactor: 3,
					Components: TempoComponentsSpec{
						Ingester: TempoComponentSpec{
							Replicas: pointer.Int32(1),
						},
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(path, 3,
					fmt.Sprintf("replica factor of %d requires at least %d ingester replicas", 3, 2),
				)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := validator.validateReplicationFactor(&test.input)
			assert.Equal(t, test.expected, errs)
		})
	}
}

package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestDefault(t *testing.T) {
	tests := []struct {
		input    *Microservices
		expected *Microservices
		name     string
	}{
		{
			name: "no action default values are provided",
			input: &Microservices{
				Spec: MicroservicesSpec{
					Retention: RetentionSpec{
						Global: RetentionConfig{
							Traces: time.Hour,
						},
					},
					StorageSize: resource.MustParse("1Gi"),
				},
			},
			expected: &Microservices{
				Spec: MicroservicesSpec{
					Retention: RetentionSpec{
						Global: RetentionConfig{
							Traces: time.Hour,
						},
					},
					StorageSize: resource.MustParse("1Gi"),
				},
			},
		},
		{
			name:  "default values are set in the webhook",
			input: &Microservices{},
			expected: &Microservices{
				Spec: MicroservicesSpec{
					Retention: RetentionSpec{
						Global: RetentionConfig{
							Traces: 48 * time.Hour,
						},
					},
					StorageSize: resource.MustParse("10Gi"),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.input.Default()
			assert.Equal(t, test.expected, test.input)
		})
	}
}

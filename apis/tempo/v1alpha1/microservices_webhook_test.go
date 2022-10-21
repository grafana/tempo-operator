package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefault(t *testing.T) {
	tests := []struct {
		input    *Microservices
		expected *Microservices
		name     string
	}{
		{
			name: "no action retention exists",
			input: &Microservices{
				Spec: MicroservicesSpec{
					Retention: RetentionSpec{
						Global: RetentionConfig{
							Traces: time.Hour,
						},
					},
				},
			},
			expected: &Microservices{
				Spec: MicroservicesSpec{
					Retention: RetentionSpec{
						Global: RetentionConfig{
							Traces: time.Hour,
						},
					},
				},
			},
		},
		{
			name:  "configure default missing retention",
			input: &Microservices{},
			expected: &Microservices{
				Spec: MicroservicesSpec{
					Retention: RetentionSpec{
						Global: RetentionConfig{
							Traces: 48 * time.Hour,
						},
					},
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

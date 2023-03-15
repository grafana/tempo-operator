package v1alpha1

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateProjectConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    ProjectConfig
		expected error
	}{
		{
			name: "valid featureGates.tlsProfile setting",
			input: ProjectConfig{
				Gates: FeatureGates{
					TLSProfile: string(TLSProfileModernType),
				},
			},
			expected: nil,
		},
		{
			name: "invalid featureGates.tlsProfile setting",
			input: ProjectConfig{
				Gates: FeatureGates{
					TLSProfile: "abc",
				},
			},
			expected: errors.New("invalid value 'abc' for setting featureGates.tlsProfile (valid values: Old, Intermediate and Modern)"),
		},
		{
			name:     "empty featureGates.tlsProfile setting",
			input:    ProjectConfig{},
			expected: errors.New("invalid value '' for setting featureGates.tlsProfile (valid values: Old, Intermediate and Modern)"),
		},
		{
			name: "invalid tempo container image",
			input: ProjectConfig{
				DefaultImages: ImagesSpec{
					Tempo: "abc@def",
				},
				Gates: FeatureGates{
					TLSProfile: "Modern",
				},
			},
			expected: errors.New("invalid value 'abc@def' for setting images.tempo"),
		},
		{
			name: "invalid tempoQuery container image",
			input: ProjectConfig{
				DefaultImages: ImagesSpec{
					TempoQuery: "abc@def",
				},
				Gates: FeatureGates{
					TLSProfile: "Modern",
				},
			},
			expected: errors.New("invalid value 'abc@def' for setting images.tempoQuery"),
		},
		{
			name: "invalid tempoGateway container image",
			input: ProjectConfig{
				DefaultImages: ImagesSpec{
					TempoGateway: "abc@def",
				},
				Gates: FeatureGates{
					TLSProfile: "Modern",
				},
			},
			expected: errors.New("invalid value 'abc@def' for setting images.tempoGateway"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.input.Validate())
		})
	}
}

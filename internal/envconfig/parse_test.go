package envconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestParsePositiveDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected metav1.Duration
		hasError bool
	}{
		{
			name:     "valid hours",
			input:    "24h",
			expected: metav1.Duration{Duration: 24 * time.Hour},
			hasError: false,
		},
		{
			name:     "valid minutes",
			input:    "30m",
			expected: metav1.Duration{Duration: 30 * time.Minute},
			hasError: false,
		},
		{
			name:     "valid seconds",
			input:    "60s",
			expected: metav1.Duration{Duration: 60 * time.Second},
			hasError: false,
		},
		{
			name:     "valid combined",
			input:    "1h30m",
			expected: metav1.Duration{Duration: 90 * time.Minute},
			hasError: false,
		},
		{
			name:     "invalid duration",
			input:    "invalid",
			expected: metav1.Duration{},
			hasError: true,
		},
		{
			name:     "negative duration",
			input:    "-1h",
			expected: metav1.Duration{},
			hasError: true,
		},
		{
			name:     "zero duration rejected",
			input:    "0s",
			expected: metav1.Duration{},
			hasError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := parsePositiveDuration(test.input)
			if test.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, result)
			}
		})
	}
}

func TestParsePodSecurityContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *corev1.PodSecurityContext
		hasError bool
	}{
		{
			name:  "valid fsGroup",
			input: `{"fsGroup": 10001}`,
			expected: &corev1.PodSecurityContext{
				FSGroup: ptr.To(int64(10001)),
			},
			hasError: false,
		},
		{
			name:  "valid runAsUser",
			input: `{"runAsUser": 1000}`,
			expected: &corev1.PodSecurityContext{
				RunAsUser: ptr.To(int64(1000)),
			},
			hasError: false,
		},
		{
			name:  "valid runAsNonRoot",
			input: `{"runAsNonRoot": true}`,
			expected: &corev1.PodSecurityContext{
				RunAsNonRoot: ptr.To(true),
			},
			hasError: false,
		},
		{
			name:     "empty object",
			input:    `{}`,
			expected: &corev1.PodSecurityContext{},
			hasError: false,
		},
		{
			name:  "multiple fields",
			input: `{"fsGroup": 10001, "runAsUser": 1000, "runAsNonRoot": true}`,
			expected: &corev1.PodSecurityContext{
				FSGroup:      ptr.To(int64(10001)),
				RunAsUser:    ptr.To(int64(1000)),
				RunAsNonRoot: ptr.To(true),
			},
			hasError: false,
		},
		{
			name:     "invalid json",
			input:    `invalid`,
			expected: nil,
			hasError: true,
		},
		{
			name:     "invalid json structure",
			input:    `{"fsGroup": "notanumber"}`,
			expected: nil,
			hasError: true,
		},
		{
			name:     "unknown field rejected",
			input:    `{"fsGroup": 10001, "unknownField": "value"}`,
			expected: nil,
			hasError: true,
		},
		{
			name:     "typo in field name rejected",
			input:    `{"fsGroups": 10001}`,
			expected: nil,
			hasError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := parsePodSecurityContext(test.input)
			if test.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, result)
			}
		})
	}
}

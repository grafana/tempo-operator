package v1alpha1

import (
	"testing"
)

func TestIngressTypeIsEnabled(t *testing.T) {
	tests := []struct {
		name        string
		ingressType IngressType
		expected    bool
	}{
		{
			name:        "unspecified type returns false",
			ingressType: IngressTypeUnspecified,
			expected:    false,
		},
		{
			name:        "none type returns false",
			ingressType: IngressTypeNone,
			expected:    false,
		},
		{
			name:        "ingress type returns true",
			ingressType: IngressTypeIngress,
			expected:    true,
		},
		{
			name:        "route type returns true",
			ingressType: IngressTypeRoute,
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ingressType.IsEnabled(); got != tt.expected {
				t.Errorf("IngressType.IsEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

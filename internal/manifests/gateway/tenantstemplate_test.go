package gateway

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

func TestTenantsTemplate(t *testing.T) {
	tests := []struct {
		name     string
		opts     options
		expected string
	}{
		{
			name: "oidc nil",
			opts: options{
				Namespace:     "default",
				Name:          "foo",
				TenantSecrets: nil,
				Tenants: &v1alpha1.TenantsSpec{
					Mode: v1alpha1.Static,
					Authentication: []v1alpha1.AuthenticationSpec{
						{
							TenantName: "dev",
							TenantID:   "abcd1",
						},
					},
				},
			},
			expected: `tenants:
- name: dev
  id: abcd1`,
		},
		{
			name: "openshift",
			opts: options{
				Namespace:     "default",
				Name:          "foo",
				TenantSecrets: nil,
				Tenants: &v1alpha1.TenantsSpec{
					Mode: v1alpha1.OpenShift,
					Authentication: []v1alpha1.AuthenticationSpec{
						{
							TenantName: "dev",
							TenantID:   "abcd1",
						},
					},
				},
			},
			expected: `tenants:
- name: dev
  id: abcd1
  openshift:
    serviceAccount: tempo-foo-serviceaccount
    redirectURL: https://foo-default.apps-crc.testing/openshift/dev/callback`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := &bytes.Buffer{}
			err := tenantsTemplate.Execute(buffer, tt.opts)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, buffer.String())
		})
	}
}

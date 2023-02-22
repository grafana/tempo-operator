package gateway

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

func TestRBACsConfig(t *testing.T) {
	tests := []struct {
		name     string
		opts     options
		expected string
	}{
		{
			name: "read write role",
			opts: options{
				Namespace:     "default",
				Name:          "foo",
				TenantSecrets: nil,
				Tenants: &v1alpha1.TenantsSpec{
					Mode: v1alpha1.Static,
					Authorization: &v1alpha1.AuthorizationSpec{
						Roles: []v1alpha1.RoleSpec{
							{
								Name:        "traces-read-write",
								Resources:   []string{"traces"},
								Tenants:     []string{"dev"},
								Permissions: []v1alpha1.PermissionType{v1alpha1.Read, v1alpha1.Write},
							},
						},
						RoleBindings: []v1alpha1.RoleBindingsSpec{
							{
								Name:  "read-write",
								Roles: []string{"traces-read-write"},
								Subjects: []v1alpha1.Subject{
									{
										Name: "user",
										Kind: "system:serviceaccount:default:dev-collector",
									},
								},
							},
						},
					},
				},
			},
			expected: `roleBindings:
- name: read-write
  roles:
  - traces-read-write

  subjects:
  - kind: system:serviceaccount:default:dev-collector
    name: user

roles:
- name: traces-read-write
  permissions:
  - read
  - write

  resources:
  - traces

  tenants:
  - dev`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := &bytes.Buffer{}
			err := rbacTemplate.Execute(buffer, tt.opts)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, buffer.String())
		})
	}
}

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
    serviceAccount: tempo-foo-gateway
    redirectURL: https://tempo-foo-gateway-default.apps-crc.testing/openshift/dev/callback`,
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

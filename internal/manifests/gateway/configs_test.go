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
				Namespace: "default",
				Name:      "foo",
				Tenants: &tenants{
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
		{
			name: "openshift mode",
			opts: options{
				Namespace: "default",
				Name:      "foo",
				Tenants: &tenants{
					Mode: v1alpha1.OpenShift,
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
			expected: ``,
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
				Namespace: "default",
				Name:      "foo",
				Tenants: &tenants{
					Mode: v1alpha1.Static,
					Authentication: []authentication{
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
			name: "with oidc",
			opts: options{
				Namespace: "default",
				Name:      "foo",
				Tenants: &tenants{
					Mode: v1alpha1.Static,
					Authentication: []authentication{
						{
							TenantName: "dev",
							TenantID:   "abcd1",
							OIDC: &v1alpha1.OIDCSpec{
								IssuerURL:     "https://something.com",
								RedirectURL:   "https://something.com/redirect",
								GroupClaim:    "groupClaim",
								UsernameClaim: "claim",
							},
							OIDCSecret: oidcSecret{
								ClientID:     "clientid",
								ClientSecret: "secret",
								IssuerCAPath: "capath",
							},
						},
					},
				},
			},
			expected: `tenants:
- name: dev
  id: abcd1
  oidc:
    clientID: clientid
    clientSecret: secret
    issuerCAPath: capath
    issuerURL: https://something.com
    redirectURL: https://something.com/redirect
    usernameClaim: claim
    groupClaim: groupClaim`,
		},
		{
			name: "with oidc from e2e Kubernetes test",
			opts: options{
				Namespace: "default",
				Name:      "foo",
				Tenants: &tenants{
					Mode: v1alpha1.Static,
					Authentication: []authentication{
						{
							TenantName: "test-oidc",
							TenantID:   "test-oidc",
							OIDC: &v1alpha1.OIDCSpec{
								IssuerURL:     "http://dex.svc:30556/dex",
								RedirectURL:   "http://tempo-foo-gateway.svc:8080/oidc/test-oidc/callback",
								UsernameClaim: "email",
							},
							OIDCSecret: oidcSecret{
								ClientID:     "test",
								ClientSecret: "ZXhhbXBsZS1hcHAtc2VjcmV0",
							},
						},
					},
				},
			},
			expected: `tenants:
- name: test-oidc
  id: test-oidc
  oidc:
    clientID: test
    clientSecret: ZXhhbXBsZS1hcHAtc2VjcmV0
    issuerURL: http://dex.svc:30556/dex
    redirectURL: http://tempo-foo-gateway.svc:8080/oidc/test-oidc/callback
    usernameClaim: email`,
		},
		{
			name: "openshift",
			opts: options{
				Namespace:  "default",
				Name:       "foo",
				BaseDomain: "apps-crc.testing",
				Tenants: &tenants{
					Mode: v1alpha1.OpenShift,
					Authentication: []authentication{
						{
							TenantName:            "dev",
							TenantID:              "abcd1",
							OpenShiftCookieSecret: "random",
						},
						{
							TenantName:            "prod",
							TenantID:              "abcd2",
							OpenShiftCookieSecret: "random2",
						},
					},
				},
			},
			expected: `tenants:
- name: dev
  id: abcd1
  openshift:
    serviceAccount: tempo-foo-gateway
    redirectURL: https://tempo-foo-gateway-default.apps-crc.testing/openshift/dev/callback
    cookieSecret: random
  opa:
    url: http://localhost:8082/v1/data/tempostack/allow
    withAccessToken: true
- name: prod
  id: abcd2
  openshift:
    serviceAccount: tempo-foo-gateway
    redirectURL: https://tempo-foo-gateway-default.apps-crc.testing/openshift/prod/callback
    cookieSecret: random2
  opa:
    url: http://localhost:8082/v1/data/tempostack/allow
    withAccessToken: true`,
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

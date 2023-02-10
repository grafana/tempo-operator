package manifests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildAll(t *testing.T) {
	objects, err := BuildAll(manifestutils.Params{
		Tempo: v1alpha1.Microservices{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "project1",
			},
			Spec: v1alpha1.MicroservicesSpec{
				Components: v1alpha1.TempoComponentsSpec{
					Gateway: v1alpha1.TempoGatewaySpec{
						Enabled: true,
					},
				},
				Tenants: &v1alpha1.TenantsSpec{
					Mode: v1alpha1.Static,
					Authentication: []v1alpha1.AuthenticationSpec{
						{
							TenantName: "test-oidc",
							TenantID:   "test-oidc",
							OIDC: &v1alpha1.OIDCSpec{
								Secret: &v1alpha1.TenantSecretSpec{
									Name: "test-oidc",
								},
								IssuerURL: "https://dex.klimlive.de/dex",
							},
						},
					},
					Authorization: &v1alpha1.AuthorizationSpec{
						RoleBindings: []v1alpha1.RoleBindingsSpec{
							{
								Name:  "test-oidc",
								Roles: []string{"read-write"},
								Subjects: []v1alpha1.Subject{
									{
										Name: "user",
										Kind: v1alpha1.User,
									},
								},
							},
						},
						Roles: []v1alpha1.RoleSpec{
							{
								Name: "read-write",
								Permissions: []v1alpha1.PermissionType{
									v1alpha1.Read, v1alpha1.Write,
								},
								Resources: []string{"logs", "traces", "metrics"},
								Tenants:   []string{"test-oidc"},
							},
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	assert.Len(t, objects, 17)
}

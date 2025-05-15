package manifests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildAll(t *testing.T) {
	objects, err := BuildAll(manifestutils.Params{
		StorageParams: manifestutils.StorageParams{
			AzureStorage: &manifestutils.AzureStorage{
				Container: "image",
			},
			GCS: &manifestutils.GCS{
				Bucket: "test",
			},
			S3: &manifestutils.S3{
				Endpoint: "https://localhost",
				Bucket:   "test",
			},
		},
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "project1",
			},
			Spec: v1alpha1.TempoStackSpec{
				Timeout: metav1.Duration{Duration: time.Second * 5},
				Template: v1alpha1.TempoTemplateSpec{
					Gateway: v1alpha1.TempoGatewaySpec{
						Enabled: true,
					},
				},
				Tenants: &v1alpha1.TenantsSpec{
					Mode: v1alpha1.ModeStatic,
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

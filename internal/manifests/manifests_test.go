package manifests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/tlsprofile"
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
		TLSProfile: tlsprofile.TLSProfileOptions{},
	})
	require.NoError(t, err)
	// 17 base objects + 9 network policies (gossip, metrics, DNS, distributor, ingester, compactor, querier, query-frontend, gateway)
	// + 5 pod disruption budgets (distributor, ingester, querier, query-frontend, gateway; no PDB for the compactor)
	assert.Len(t, objects, 31)
}

func TestBuildAllPodAntiAffinity(t *testing.T) {
	// a component-specific anti affinity requiring, rather than preferring, one pod per node
	podAntiAffinity := func(component string) *corev1.PodAntiAffinity {
		return &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
				LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{
					"app.kubernetes.io/component": component,
				}},
				TopologyKey: corev1.LabelHostname,
			}},
		}
	}
	componentSpec := func(component string) v1alpha1.TempoComponentSpec {
		return v1alpha1.TempoComponentSpec{PodAntiAffinity: podAntiAffinity(component)}
	}

	objects, err := BuildAll(manifestutils.Params{
		StorageParams: manifestutils.StorageParams{S3: &manifestutils.S3{Endpoint: "https://localhost", Bucket: "test"}},
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "project1"},
			Spec: v1alpha1.TempoStackSpec{
				Timeout: metav1.Duration{Duration: time.Second * 5},
				Template: v1alpha1.TempoTemplateSpec{
					Distributor: v1alpha1.TempoDistributorSpec{
						TempoComponentSpec: componentSpec("distributor"),
					},
					Ingester:  componentSpec("ingester"),
					Querier:   componentSpec("querier"),
					Compactor: componentSpec("compactor"),
					QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
						TempoComponentSpec: componentSpec("query-frontend"),
					},
					MetricsGenerator: v1alpha1.TempoMetricsGeneratorSpec{
						Enabled:            true,
						TempoComponentSpec: componentSpec("metrics-generator"),
					},
					Gateway: v1alpha1.TempoGatewaySpec{
						Enabled:            true,
						TempoComponentSpec: componentSpec("gateway"),
					},
				},
				Tenants: &v1alpha1.TenantsSpec{
					Mode: v1alpha1.ModeStatic,
					Authentication: []v1alpha1.AuthenticationSpec{{
						TenantName: "test-oidc",
						TenantID:   "test-oidc",
						OIDC: &v1alpha1.OIDCSpec{
							Secret:    &v1alpha1.TenantSecretSpec{Name: "test-oidc"},
							IssuerURL: "https://dex.klimlive.de/dex",
						},
					}},
					Authorization: &v1alpha1.AuthorizationSpec{
						RoleBindings: []v1alpha1.RoleBindingsSpec{{
							Name:     "test-oidc",
							Roles:    []string{"read-write"},
							Subjects: []v1alpha1.Subject{{Name: "user", Kind: v1alpha1.User}},
						}},
						Roles: []v1alpha1.RoleSpec{{
							Name:        "read-write",
							Permissions: []v1alpha1.PermissionType{v1alpha1.Read, v1alpha1.Write},
							Resources:   []string{"logs", "traces", "metrics"},
							Tenants:     []string{"test-oidc"},
						}},
					},
				},
			},
		},
		TLSProfile: tlsprofile.TLSProfileOptions{},
	})
	require.NoError(t, err)

	// every component must honor its own anti affinity, otherwise the field is silently ignored
	components := map[string]bool{}
	for _, obj := range objects {
		var template *corev1.PodTemplateSpec
		switch workload := obj.(type) {
		case *appsv1.Deployment:
			template = &workload.Spec.Template
		case *appsv1.StatefulSet:
			template = &workload.Spec.Template
		default:
			continue
		}

		component := template.Labels["app.kubernetes.io/component"]
		components[component] = true
		assert.Equal(t, podAntiAffinity(component), template.Spec.Affinity.PodAntiAffinity, "component %s", component)
	}

	assert.Equal(t, map[string]bool{
		"distributor": true, "ingester": true, "querier": true, "compactor": true,
		"query-frontend": true, "metrics-generator": true, "gateway": true,
	}, components)
}

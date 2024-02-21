package monolithic

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

func TestGateway(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				OpenShift: configv1alpha1.OpenShiftFeatureGates{
					ServingCertsService: true,
				},
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Multitenancy: &v1alpha1.MonolithicMultitenancySpec{
					Enabled: true,
					TenantsSpec: v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeOpenShift,
						Authentication: []v1alpha1.AuthenticationSpec{
							{
								TenantName: "dev",
								TenantID:   "1610b0c3-c509-4592-a256-a1871353dbfa",
							},
							{
								TenantName: "prod",
								TenantID:   "1610b0c3-c509-4592-a256-a1871353dbfb",
							},
						},
					},
				},
			},
		},
	}
	objs, annotations, err := BuildGatewayObjects(opts)
	require.NoError(t, err)
	require.Len(t, objs, 4)

	require.Contains(t, annotations, "tempo.grafana.com/rbacConfig.hash")
	require.Contains(t, annotations, "tempo.grafana.com/tenantsConfig.hash")
}

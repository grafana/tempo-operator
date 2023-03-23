package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

func TestPatchOPAContainer(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simplest",
			Namespace: "observability",
		},
		Spec: v1alpha1.TempoStackSpec{
			Tenants: &v1alpha1.TenantsSpec{
				Mode: v1alpha1.OpenShift,
				Authentication: []v1alpha1.AuthenticationSpec{
					{
						TenantName: "dev",
						TenantID:   "abcd1",
					},
					{
						TenantName: "prod",
						TenantID:   "abcd2",
					},
				},
			},
		},
	}
	dep, err := patchOCPOPAContainer(tempo, &appsv1.Deployment{})
	require.NoError(t, err)
	require.Equal(t, 1, len(dep.Spec.Template.Spec.Containers))
	assert.Equal(t, []string{
		"--log.level=warn",
		"--opa.admin-groups=system:cluster-admins,cluster-admin,dedicated-admin",
		"--web.listen=:8082", "--web.internal.listen=:8083",
		"--web.healthchecks.url=http://localhost:8082",
		"--opa.package=tempostack",
		"--openshift.mappings=dev=tempo.grafana.com",
		"--openshift.mappings=prod=tempo.grafana.com",
	}, dep.Spec.Template.Spec.Containers[0].Args)
}

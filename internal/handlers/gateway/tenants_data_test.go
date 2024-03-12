package gateway

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestGetGatewayTenantsData(t *testing.T) {
	namespace := createNamespace(t, "test-tenants-secret")
	assert.Equal(t, "test-tenants-secret", namespace.Name)

	nsn := types.NamespacedName{Name: "tempo-simplest-gateway", Namespace: namespace.Name}
	secret := createSecret(t, nsn, map[string]string{
		"tenants.yaml": `tenants:
- name: dev
  id: 1610b0c3-c509-4592-a256-a1871353dbfa
  openshift:
    serviceAccount: tempo-simplest-gateway
    redirectURL: https://tempo-simplest-gateway-observability.apps-crc.testing/openshift/dev/callback
    cookieSecret: f6UiBYSteEXdD3SrJRfBj3XzefPrRJmi
  opa:
    url: http://localhost:8082/v1/data/tempostack/allow
    withAccessToken: true
- name: prod
  id: 1610b0c3-c509-4592-a256-a1871353dbfb
  openshift:
    serviceAccount: tempo-simplest-gateway
    redirectURL: https://tempo-simplest-gateway-observability.apps-crc.testing/openshift/prod/callback
    cookieSecret: ZjMdgk8btNHekSBg87WArUqopjlaW8BL
  opa:
    url: http://localhost:8082/v1/data/tempostack/allow
    withAccessToken: true`,
	})
	assert.Equal(t, "tempo-simplest-gateway", secret.Name)

	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simplest",
			Namespace: namespace.Name,
		},
	}
	tenantsData, err := GetGatewayTenantsData(context.Background(), k8sClient, tempo.Namespace, tempo.Name)
	require.NoError(t, err)
	require.Equal(t, 2, len(tenantsData))
	assert.Equal(t, &manifestutils.GatewayTenantsData{
		TenantName:            "dev",
		OpenShiftCookieSecret: "f6UiBYSteEXdD3SrJRfBj3XzefPrRJmi",
	}, tenantsData[0])
}

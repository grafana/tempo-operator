package networking

import (
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkPolicyDistributor(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myinstance",
			Namespace: "something",
		},
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				Gateway: v1alpha1.TempoGatewaySpec{
					Enabled: true,
				},
				QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
					JaegerQuery: v1alpha1.JaegerQuerySpec{
						Enabled: true,
					},
				},
			},
		},
	}

	componentName := manifestutils.DistributorComponentName
	np := generatePolicyFor(tempo, componentName)

	require.NotNil(t, np)

	assert.Equal(t, np.ObjectMeta.Name, naming.Name(componentName, tempo.Name))
	assert.Equal(t, np.ObjectMeta.Namespace, tempo.Namespace)
	assert.Equal(t, np.Spec.PodSelector, manifestutils.ComponentLabels(componentName, tempo.Name))
	assert.Len(t, np.Spec.PolicyTypes, 2)
}

func TestReverseRelations(t *testing.T) {
	port := networkingv1.NetworkPolicyPort{}
	relations := map[string]map[string][]networkingv1.NetworkPolicyPort{
		"A": {
			"B": {port},
			"C": {port},
		},
		"B": {
			"C": {port},
		},
	}

	expected := map[string]map[string][]networkingv1.NetworkPolicyPort{
		"B": {
			"A": {port},
		},
		"C": {
			"A": {port},
			"B": {port},
		},
	}

	reversed := reverseRelations(relations)
	require.Equal(t, expected, reversed)
}

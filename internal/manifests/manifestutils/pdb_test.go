package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func TestNewPodDisruptionBudget(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "project1",
		},
	}

	pdb := NewPodDisruptionBudget(tempo, DistributorComponentName)

	require.NotNil(t, pdb)
	assert.Equal(t, "PodDisruptionBudget", pdb.Kind)
	assert.Equal(t, "policy/v1", pdb.APIVersion)
	assert.Equal(t, "tempo-test-distributor", pdb.Name)
	assert.Equal(t, "project1", pdb.Namespace)

	labels := map[string]string(ComponentLabels(DistributorComponentName, "test"))
	assert.Equal(t, labels, pdb.Labels)

	require.NotNil(t, pdb.Spec.Selector)
	assert.Equal(t, labels, pdb.Spec.Selector.MatchLabels)

	require.NotNil(t, pdb.Spec.MaxUnavailable)
	assert.Equal(t, intstr.FromInt32(1), *pdb.Spec.MaxUnavailable)
	assert.Nil(t, pdb.Spec.MinAvailable)
}

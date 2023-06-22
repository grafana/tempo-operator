package serviceaccount

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildDefaultServiceAccount(t *testing.T) {
	serviceAccount := BuildDefaultServiceAccount(v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns1",
		},
	})

	labels := manifestutils.ComponentLabels("serviceaccount", "test")
	require.NotNil(t, serviceAccount)
	assert.Equal(t, &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-test",
			Namespace: "ns1",
			Labels:    labels,
		},
	}, serviceAccount)
}

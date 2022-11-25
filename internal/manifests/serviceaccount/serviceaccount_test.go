package serviceaccount

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildDefaultServiceAccount(t *testing.T) {
	serviceAccount := BuildDefaultServiceAccount(v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns1",
		},
	})

	labels := manifestutils.ComponentLabels("serviceaccount", "test")
	require.NotNil(t, serviceAccount)
	assert.Equal(t, &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-test-serviceaccount",
			Namespace: "ns1",
			Labels:    labels,
		},
	}, serviceAccount)
}

func TestServiceAccountName(t *testing.T) {
	serviceAccountName1 := ServiceAccountName(v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns1",
		},
	})
	assert.Equal(t, "tempo-test-serviceaccount", serviceAccountName1)

	serviceAccountName2 := ServiceAccountName(v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns1",
		},
		Spec: v1alpha1.MicroservicesSpec{
			ServiceAccount: "existing-sa",
		},
	})
	assert.Equal(t, "existing-sa", serviceAccountName2)
}

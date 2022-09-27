package manifests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/os-observability/tempo-operator/api/v1alpha1"
)

func TestBuildAll(t *testing.T) {
	objects, err := BuildAll(Params{Tempo: v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "project1",
		},
	}})
	require.NoError(t, err)
	assert.Equal(t, 3, len(objects))
}

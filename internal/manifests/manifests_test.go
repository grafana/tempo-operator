package manifests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildAll(t *testing.T) {
	objects, err := BuildAll(manifestutils.Params{Tempo: v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "project1",
		},
	}})
	require.NoError(t, err)
	assert.Len(t, objects, 13)
}

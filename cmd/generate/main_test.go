package generate

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configtempov1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests"
)

func TestBuild(t *testing.T) {
	ctrlConfig := configtempov1alpha1.ProjectConfig{
		DefaultImages: v1alpha1.ImagesSpec{
			Tempo:      "tempo-image",
			TempoQuery: "tempo-query-image",
		},
	}
	params := manifests.Params{
		StorageParams: manifests.StorageParams{
			S3: manifests.S3{},
		},
	}

	objects, err := build(ctrlConfig, params)
	require.NoError(t, err)
	require.GreaterOrEqual(t, 14, len(objects))
}

func TestYAMLEncoding(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-config-map",
		},
		Data: map[string]string{
			"tempo.yaml": "ingester:\n  setting: a\n",
		},
	}

	var buf bytes.Buffer
	err := toYAMLManifest(scheme, []client.Object{&cm}, &buf)
	require.NoError(t, err)
	require.YAMLEq(t, `---
apiVersion: v1
data:
  tempo.yaml: |
    ingester:
      setting: a
kind: ConfigMap
metadata:
  creationTimestamp: null
  name: test-config-map
`, buf.String())
}

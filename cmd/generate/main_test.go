package generate

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/cmd/root"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestBuild(t *testing.T) {
	params := manifestutils.Params{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo:           "tempo-image",
				TempoQuery:      "tempo-query-image",
				TempoGateway:    "tempo-gateway-image",
				TempoGatewayOpa: "tempo-gateway-opa-image",
			},
		},
		StorageParams: manifestutils.StorageParams{
			AzureStorage: &manifestutils.AzureStorage{},
			GCS:          &manifestutils.GCS{},
			S3:           &manifestutils.S3{},
		},
	}

	objects, err := build(params)
	require.NoError(t, err)
	require.Equal(t, 14, len(objects))
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
  name: test-config-map
`, buf.String())
}

func TestGenerateCmdReadFromStdin(t *testing.T) {
	c := root.NewRootCommand()
	c.AddCommand(NewGenerateCommand())

	cr := `
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplest
spec:
  images:
    tempo: docker.io/grafana/tempo:x.y.z
    tempoQuery: docker.io/grafana/tempo-query:x.y.z
    tempoGateway: quay.io/observatorium/api
    tempoGatewayOpa: quay.io/observatorium/opa-openshift
  storage:
    secret:
      name: minio-test
      type: s3
  storageSize: 1Gi
`
	c.SetIn(strings.NewReader(cr))

	out := &strings.Builder{}
	c.SetOut(out)
	c.SetErr(out)

	c.SetArgs([]string{"generate"})
	_, err := c.ExecuteC()
	require.NoError(t, err)

	require.Contains(t, out.String(), `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/component: distributor
    app.kubernetes.io/instance: simplest
    app.kubernetes.io/managed-by: tempo-operator
    app.kubernetes.io/name: tempo
  name: tempo-simplest-distributor
`)
}

func TestGenerateCmdReadFromFile(t *testing.T) {
	c := root.NewRootCommand()
	c.AddCommand(NewGenerateCommand())

	out := &strings.Builder{}
	c.SetOut(out)
	c.SetErr(out)

	c.SetArgs([]string{"generate", "--cr", "testdata/cr.yaml"})
	_, err := c.ExecuteC()
	require.NoError(t, err)

	require.Contains(t, out.String(), `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/component: distributor
    app.kubernetes.io/instance: simplest
    app.kubernetes.io/managed-by: tempo-operator
    app.kubernetes.io/name: tempo
  name: tempo-simplest-distributor
`)
}

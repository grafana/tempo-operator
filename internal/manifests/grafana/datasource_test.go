package grafana

import (
	"testing"

	grafanav1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildGrafanaDatasource(t *testing.T) {
	datasource := BuildGrafanaDatasource(manifestutils.Params{Tempo: v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "tempo",
		},
		Spec: v1alpha1.TempoStackSpec{},
	}})
	labels := manifestutils.CommonLabels("test")

	require.NotNil(t, datasource)
	require.Equal(t, &grafanav1.GrafanaDatasource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "grafana.integreatly.org/v1beta1",
			Kind:       "GrafanaDatasource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "tempo",
			Labels:    labels,
		},
		Spec: grafanav1.GrafanaDatasourceSpec{
			Datasource: &grafanav1.GrafanaDatasourceInternal{
				Access: "proxy",
				Name:   "test",
				Type:   "tempo",
				URL:    "http://tempo-test-query-frontend.tempo.svc.cluster.local:3200",
			},
			InstanceSelector:          &metav1.LabelSelector{},
			AllowCrossNamespaceImport: ptr.To(true),
		},
	}, datasource)
}

package monolithic

import (
	"testing"

	grafanav1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func TestBuildGrafanaDatasource(t *testing.T) {
	opts := Options{
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Observability: &v1alpha1.MonolithicObservabilitySpec{
					Grafana: &v1alpha1.MonolithicObservabilityGrafanaSpec{
						DataSource: &v1alpha1.MonolithicObservabilityGrafanaDataSourceSpec{
							Enabled: true,
							InstanceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"key": "value"},
							},
						},
					},
				},
			},
		},
	}
	datasource := BuildGrafanaDatasource(opts)

	labels := ComponentLabels("tempo", "sample")
	require.Equal(t, &grafanav1.GrafanaDatasource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "grafana.integreatly.org/v1beta1",
			Kind:       "GrafanaDatasource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "sample",
			Labels:    labels,
		},
		Spec: grafanav1.GrafanaDatasourceSpec{
			Datasource: &grafanav1.GrafanaDatasourceInternal{
				Name:   "sample",
				Type:   "tempo",
				Access: "proxy",
				URL:    "http://tempo-sample.default.svc.cluster.local:3200",
			},
			InstanceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"key": "value"},
			},
			AllowCrossNamespaceImport: ptr.To(true),
		},
	}, datasource)
}

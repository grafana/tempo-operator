package grafana

import (
	"fmt"

	grafanav1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

// BuildGrafanaDatasource creates a data source for Grafana Tempo.
func BuildGrafanaDatasource(params manifestutils.Params) *grafanav1.GrafanaDatasource {
	tempo := params.Tempo
	labels := manifestutils.CommonLabels(tempo.Name)
	url := fmt.Sprintf("http://%s:%d", naming.ServiceFqdn(tempo.Namespace, tempo.Name, manifestutils.QueryFrontendComponentName), manifestutils.PortHTTPServer)
	return NewGrafanaDatasource(tempo.Namespace, tempo.Name, labels, url, tempo.Spec.Observability.Grafana.InstanceSelector)
}

// NewGrafanaDatasource creates a data source for Grafana Tempo.
func NewGrafanaDatasource(
	namespace string,
	name string,
	labels labels.Set,
	url string,
	instanceSelector metav1.LabelSelector,
) *grafanav1.GrafanaDatasource {
	return &grafanav1.GrafanaDatasource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: grafanav1.GroupVersion.String(),
			Kind:       "GrafanaDatasource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: grafanav1.GrafanaDatasourceSpec{
			Datasource: &grafanav1.GrafanaDatasourceInternal{
				Name:   name,
				Type:   "tempo",
				Access: "proxy",
				URL:    url,
			},

			// InstanceSelector is a required field in the spec
			InstanceSelector: &instanceSelector,

			// Allow using this datasource from Grafana instances in other namespaces
			AllowCrossNamespaceImport: ptr.To(true),
		},
	}
}

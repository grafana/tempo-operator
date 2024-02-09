package monolithic

import (
	"fmt"

	grafanav1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/internal/manifests/grafana"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

// BuildGrafanaDatasource create a Grafana data source.
func BuildGrafanaDatasource(opts Options) *grafanav1.GrafanaDatasource {
	tempo := opts.Tempo
	labels := ComponentLabels(manifestutils.TempoMonolithComponentName, tempo.Name)
	url := fmt.Sprintf("http://%s:%d", naming.ServiceFqdn(tempo.Namespace, tempo.Name, manifestutils.TempoMonolithComponentName), manifestutils.PortHTTPServer)
	instanceSelector := ptr.Deref(tempo.Spec.Observability.Grafana.DataSource.InstanceSelector, metav1.LabelSelector{})
	return grafana.NewGrafanaDatasource(tempo.Namespace, tempo.Name, labels, url, instanceSelector)
}

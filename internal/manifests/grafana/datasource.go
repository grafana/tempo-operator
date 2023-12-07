package grafana

import (
	"encoding/json"
	"fmt"

	grafanav1 "github.com/grafana-operator/grafana-operator/v5/api/v1beta1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildGrafanaDatasource creates a Datasource for Grafana Tempo.
func BuildGrafanaDatasource(params manifestutils.Params) (*grafanav1.GrafanaDatasource, error) {
	var tlsSkipVerify = true
	var component = manifestutils.QueryFrontendComponentName

	if params.Tempo.Spec.Template.Gateway.Enabled {
		component = manifestutils.GatewayComponentName
	}

	return &grafanav1.GrafanaDatasource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(component, params.Tempo.Name),
			Namespace: params.Tempo.Namespace,
			Labels:    manifestutils.CommonLabels(params.Tempo.Name),
		},
		Spec: grafanav1.GrafanaDatasourceSpec{
			Datasource: &grafanav1.GrafanaDatasourceInternal{
				Access:   "proxy",
				Name:     params.Tempo.Name,
				Type:     "tempo",
				URL:      fmt.Sprintf("https://%s:%d", naming.ServiceFqdn(params.Tempo.Namespace, params.Tempo.Name, component), manifestutils.PortHTTPServer),
				JSONData: json.RawMessage(fmt.Sprintf(`{"tlsSkipVerify": %t}`, tlsSkipVerify)),
			},
			InstanceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "tempo-operator",
				},
			},
		},
	}, nil
}

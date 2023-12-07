package grafana

import (
	grafanav1 "github.com/grafana-operator/grafana-operator/v5/api/v1beta1"
	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// BuildGrafanaDatasource creates a Datasource for Grafana Tempo.
func BuildGrafanaDatasource(params manifestutils.Params) (*grafanav1.GrafanaDatasource, error) {
	var tlsSkipVerify = true
	var url = naming.Name(manifestutils.QueryFrontendComponentName, name)
	var component = manifestutils.QueryFrontendComponentName

	if params.Tempo.Spec.Template.Gateway.Enabled {
		url := naming.Name(manifestutils.GatewayComponentName, name)
		component := manifestutils.GatewayComponentName
	}

	return &grafanav1.GrafanaDatasource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(component, params.Tempo.Name),
			Namespace: params.Tempo.Namespace,
			Labels:    manifestutils.CommonLabels(params.Tempo.Name),
		},
		Spec: grafanav1.GrafanaDatasourceSpec{
			DatasourceSpec: grafanav1.DatasourceSpec{
				Access: "proxy",
				Name:  params.Tempo.Name,
				Type: "tempo",
				URL:  fmt.Sprintf("https://%s:%d", naming.ServiceFqdn(params.Tempo.Namespace, params.Tempo.Name, url), manifestutils.PortHTTPServer),
				JSONData: json.RawMessage(fmt.Sprintf(`{"tlsSkipVerify": %t}`, tlsSkipVerify)),
			},
			InstanceSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "tempo-operator",
				},
			}
		}
	}
}

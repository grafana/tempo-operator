package datasource

import (
	grafanav1 "github.com/grafana-operator/grafana-operator/v5/api/v1beta1"
	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// BuildGrafanaDatasource creates a Datasource for Grafana Tempo.
func BuildGrafanaDatasource(string namespace, string name) (*grafanav1.GrafanaDatasource, error) {
	var tlsSkipVerify = true
	var url = naming.Name(manifestutils.QueryFrontendComponentName, name)

	if params.Tempo.Spec.Template.Gateway.Enabled {
		url := naming.Name(manifestutils.GatewayComponentName, name)
	}

	return &grafanav1.GrafanaDatasource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(component, name),
			Namespace: namespace,,
			Labels:    manifestutils.CommonLabels(name),
		},
		Spec: grafanav1.GrafanaDatasourceSpec{
			DatasourceSpec: grafanav1.DatasourceSpec{
				Access: "proxy",
				Name:  name,
				Type: "tempo",
				URL:  "http://"+url+namespace+".svc:"+manifestutils.PortHTTPServer,
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

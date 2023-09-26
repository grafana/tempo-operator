package grafana

import (
	grafanav1 "github.com/grafana-operator/grafana-operator/v5/api/v1beta1"
	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
)

// Datasource creates a Datasource for Grafana Tempo.
func Datasource(featureGates configv1alpha1.FeatureGates, namespace string) *grafanav1.GrafanaDatasource {
	return &grafanav1.GrafanaDatasource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo",
			Namespace: namespace,
		},
		Spec: grafanav1.GrafanaDatasourceSpec{
			DatasourceSpec: grafanav1.DatasourceSpec{
				Access: "proxy",
				Name:  "Tempo",
				Type: "tempo",
				// TODO: Change URL
				URL:  "http://tempo-tempo-gateway:8080",
			},
			InstanceSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "tempo-operator",
				},
			}
		}
	}
}

package grafana

import (
	grafanav1 "github.com/grafana-operator/grafana-operator/v5/api/v1beta1"
	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
)

// Datasource creates a Datasource for Grafana Tempo.
func Datasource(featureGates configv1alpha1.FeatureGates, namespace string) *grafanav1.GrafanaDatasource {

}

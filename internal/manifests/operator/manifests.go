package operator

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/operator/prometheus"
	"go.searchlight.dev/grafana-operator"

)
// BuildAll generates manifests for all enabled features of the operator.
func BuildAll(featureGates configv1alpha1.FeatureGates, namespace string) ([]client.Object, error) {
	var manifests []client.Object

	if featureGates.Observability.Metrics.CreateServiceMonitors {
		manifests = append(manifests, prometheus.ServiceMonitor(featureGates, namespace))
	}

	if featureGates.Observability.Metrics.CreatePrometheusRules {
		prometheusRule, err := prometheus.PrometheusRule(namespace)
		if err != nil {
			return nil, err
		}

		manifests = append(manifests, prometheusRule)
	}

	if featureGates.Observability.Datasources.CreateDatasources{
		datasources, err := 
		if err != nil {
			return nil, err
		}

		manifests = append(manifests, datasources)
	}

	return manifests, nil
}

package operator

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/networking"
	"github.com/grafana/tempo-operator/internal/manifests/operator/prometheus"
)

// BuildAll generates manifests for all enabled features of the operator.
func BuildAll(featureGates configv1alpha1.FeatureGates, namespace, k8sVersion string) ([]client.Object, error) {
	var manifests []client.Object

	if featureGates.Observability.Metrics.CreateServiceMonitors {
		manifests = append(manifests, prometheus.ServiceMonitor(featureGates, namespace))
	}

	if featureGates.NetworkPolicies {
		manifests = append(manifests, networking.GenerateOperatorPolicies(namespace)...)
	}

	if featureGates.Observability.Metrics.CreatePrometheusRules {
		prometheusRule, err := prometheus.PrometheusRule(namespace)
		if err != nil {
			return nil, err
		}

		manifests = append(manifests, prometheusRule)
	}

	return manifests, nil
}

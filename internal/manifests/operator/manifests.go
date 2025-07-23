package operator

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/util/version"

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

	discovered, err := version.Parse(k8sVersion)
	if err != nil {
		return nil, err
	}

	const minVersion = "1.31" // NOTE: OpenShift 4.19.
	minimum := version.MustParse(minVersion)

	if featureGates.NetworkPolicies && discovered.AtLeast(minimum) {
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

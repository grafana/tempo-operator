package operatormanifests

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
)

// CommonLabels returns the common labels for operator components.
func CommonLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "tempo-operator",
		"app.kubernetes.io/part-of":    "tempo-operator",
		"app.kubernetes.io/managed-by": "operator-lifecycle-manager",
		"control-plane":                "controller-manager",
	}
}

// BuildAll generates manifests for all enabled features of the operator.
func BuildAll(featureGates configv1alpha1.FeatureGates, namespace string) ([]client.Object, error) {
	var manifests []client.Object

	if featureGates.Observability.Metrics.CreateServiceMonitors {
		manifests = append(manifests, serviceMonitor(featureGates, namespace))
	}

	if featureGates.Observability.Metrics.CreatePrometheusRules {
		prometheusRule, err := prometheusRule(namespace)
		if err != nil {
			return nil, err
		}

		manifests = append(manifests, prometheusRule)
	}

	return manifests, nil
}

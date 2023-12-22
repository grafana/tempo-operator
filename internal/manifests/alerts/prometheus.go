package alerts

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

const (
	// RunbookDefaultURL is the default url for the documentation of the Prometheus alerts.
	RunbookDefaultURL = "https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md"
)

// BuildPrometheusRule returns a list of k8s objects for Tempo PrometheusRule.
func BuildPrometheusRule(stackName, namespace string) ([]client.Object, error) {
	prometheusRule, err := newPrometheusRule(stackName, namespace)
	if err != nil {
		return nil, err
	}

	return []client.Object{
		prometheusRule,
	}, nil
}

func newPrometheusRule(stackName, namespace string) (*monitoringv1.PrometheusRule, error) {
	alertOpts := Options{
		RunbookURL: RunbookDefaultURL,
		Cluster:    stackName,
		Namespace:  namespace,
	}

	spec, err := build(alertOpts)
	if err != nil {
		return nil, err
	}

	return &monitoringv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			Kind:       monitoringv1.PrometheusRuleKind,
			APIVersion: monitoringv1.SchemeGroupVersion.String(),
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.PrometheusRuleName(stackName),
			Namespace: namespace,
			Labels: map[string]string{
				"openshift.io/prometheus-rule-evaluation-scope": "leaf-prometheus",
			},
		},
		Spec: *spec,
	}, nil
}

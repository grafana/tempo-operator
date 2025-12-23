package alerts

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

const (
	// RunbookDefaultURL is the default url for the documentation of the Prometheus alerts.
	RunbookDefaultURL = "https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md"
)

// BuildPrometheusRule returns a list of k8s objects for Tempo PrometheusRule.
func BuildPrometheusRule(params manifestutils.Params) ([]client.Object, error) {
	labels := manifestutils.CommonLabels(params.Tempo.Name)
	extraLabels := params.Tempo.Spec.Observability.Metrics.ExtraPrometheusRuleLabels
	prometheusRule, err := NewPrometheusRule(params.Tempo.Name, params.Tempo.Namespace, k8slabels.Merge(extraLabels, labels))
	if err != nil {
		return nil, err
	}

	return []client.Object{
		prometheusRule,
	}, nil
}

// NewPrometheusRule build a PrometheusRule.
func NewPrometheusRule(stackName, namespace string, labels k8slabels.Set) (*monitoringv1.PrometheusRule, error) {
	promRulelabels := map[string]string{
		"openshift.io/prometheus-rule-evaluation-scope": "leaf-prometheus",
	}

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
			Labels:    k8slabels.Merge(labels, promRulelabels),
		},
		Spec: *spec,
	}, nil
}

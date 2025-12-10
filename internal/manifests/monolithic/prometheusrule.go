package monolithic

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"

	"github.com/grafana/tempo-operator/internal/manifests/alerts"
)

// BuildPrometheusRules creates a PrometheusRule object.
func BuildPrometheusRules(opts Options) (*monitoringv1.PrometheusRule, error) {
	tempo := opts.Tempo
	labels := CommonLabels(opts.Tempo.Name)
	if opts.Tempo.Spec.Observability != nil &&
		opts.Tempo.Spec.Observability.Metrics != nil &&
		opts.Tempo.Spec.Observability.Metrics.PrometheusRules != nil {
		labels = k8slabels.Merge(opts.Tempo.Spec.Observability.Metrics.PrometheusRules.ExtraLabels, labels)
	}
	return alerts.NewPrometheusRule(tempo.Name, tempo.Namespace, labels)
}

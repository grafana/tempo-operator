package monolithic

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/internal/manifests/alerts"
)

// BuildPrometheusRules creates PrometheusRule objects.
func BuildPrometheusRules(opts Options) ([]client.Object, error) {
	tempo := opts.Tempo
	return alerts.BuildPrometheusRule(tempo.Name, tempo.Namespace)
}

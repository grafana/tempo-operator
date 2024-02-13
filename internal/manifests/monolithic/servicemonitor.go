package monolithic

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/servicemonitor"
)

// BuildServiceMonitor create a ServiceMonitor.
func BuildServiceMonitor(opts Options) *monitoringv1.ServiceMonitor {
	tempo := opts.Tempo
	labels := ComponentLabels(manifestutils.TempoMonolithComponentName, tempo.Name)
	return servicemonitor.NewServiceMonitor(tempo.Namespace, tempo.Name, labels, false, manifestutils.TempoMonolithComponentName, manifestutils.HttpPortName)
}

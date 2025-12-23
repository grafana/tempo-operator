package monolithic

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/servicemonitor"
)

// BuildServiceMonitor creates a ServiceMonitor.
func BuildServiceMonitor(opts Options) *monitoringv1.ServiceMonitor {
	tempo := opts.Tempo
	extraLabels := labels.Set{}
	if opts.Tempo.Spec.Observability != nil &&
		opts.Tempo.Spec.Observability.Metrics != nil &&
		opts.Tempo.Spec.Observability.Metrics.ServiceMonitors != nil {
		extraLabels = opts.Tempo.Spec.Observability.Metrics.ServiceMonitors.ExtraLabels
	}

	if tempo.Spec.Multitenancy.IsGatewayEnabled() {
		labels := ComponentLabels(manifestutils.GatewayComponentName, tempo.Name)
		return servicemonitor.NewServiceMonitor(tempo.Namespace, tempo.Name, labels, extraLabels, opts.CtrlConfig.Gates.HTTPEncryption,
			manifestutils.TempoMonolithComponentName,
			[]string{
				manifestutils.GatewayInternalHttpPortName,
				manifestutils.HttpPortName,
			})
	} else {
		labels := ComponentLabels(manifestutils.TempoMonolithComponentName, tempo.Name)
		return servicemonitor.NewServiceMonitor(
			tempo.Namespace, tempo.Name, labels, extraLabels, false, manifestutils.TempoMonolithComponentName, []string{manifestutils.HttpPortName})
	}
}

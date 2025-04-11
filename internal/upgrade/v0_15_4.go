package upgrade

import (
	"context"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

// Switch thanos-querier port to tenancy-enabled port.
func upgrade0_15_4(ctx context.Context, u Upgrade, tempo *v1alpha1.TempoStack) error {
	if tempo.Spec.Template.QueryFrontend.JaegerQuery.MonitorTab.PrometheusEndpoint == "https://thanos-querier.openshift-monitoring.svc.cluster.local:9091" {
		tempo.Spec.Template.QueryFrontend.JaegerQuery.MonitorTab.PrometheusEndpoint = "https://thanos-querier.openshift-monitoring.svc.cluster.local:9092"
	}
	return nil
}

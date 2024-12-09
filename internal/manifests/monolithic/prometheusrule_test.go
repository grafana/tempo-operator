package monolithic

import (
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func TestBuildPrometheusRules(t *testing.T) {
	opts := Options{
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
		},
	}
	objects, err := BuildPrometheusRules(opts)
	require.NoError(t, err)

	rules := objects[0].(*monitoringv1.PrometheusRule)
	require.Equal(t, "sample-prometheus-rule", rules.Name)
	require.Len(t, rules.Spec.Groups, 2)
	require.Len(t, rules.Spec.Groups[0].Rules, 14) // alert rules
	require.Len(t, rules.Spec.Groups[1].Rules, 6)  // recording rules
	require.Equal(t, "cluster_namespace_job_route:tempo_request_duration_seconds:99quantile{cluster=\"sample\", namespace=\"default\", route!~\"metrics|/frontend.Frontend/Process|debug_pprof\"} > 3\n", rules.Spec.Groups[0].Rules[0].Expr.StrVal)
}

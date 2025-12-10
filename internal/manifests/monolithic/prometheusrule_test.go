package monolithic

import (
	"testing"

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
	rules, err := BuildPrometheusRules(opts)
	require.NoError(t, err)

	require.Equal(t, "sample-prometheus-rule", rules.Name)
	require.Len(t, rules.Spec.Groups, 2)
	require.Len(t, rules.Spec.Groups[0].Rules, 14) // alert rules
	require.Len(t, rules.Spec.Groups[1].Rules, 6)  // recording rules
	require.Equal(t, "cluster_namespace_job_route:tempo_request_duration_seconds:99quantile{cluster=\"sample\", namespace=\"default\", route!~\"metrics|/frontend.Frontend/Process|debug_pprof\"} > 3\n", rules.Spec.Groups[0].Rules[0].Expr.StrVal)
}

func TestBuildPrometheusRulesWithExtraLabels(t *testing.T) {
	extraLabels := map[string]string{
		"monitoring": "enabled",
		"team":       "platform",
	}
	opts := Options{
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Observability: &v1alpha1.MonolithicObservabilitySpec{
					Metrics: &v1alpha1.MonolithicObservabilityMetricsSpec{
						PrometheusRules: &v1alpha1.MonolithicObservabilityMetricsPrometheusRulesSpec{
							Enabled:     true,
							ExtraLabels: extraLabels,
						},
					},
				},
			},
		},
	}
	rules, err := BuildPrometheusRules(opts)
	require.NoError(t, err)

	expectedLabels := map[string]string{
		"app.kubernetes.io/instance":                    "sample",
		"app.kubernetes.io/managed-by":                  "tempo-operator",
		"app.kubernetes.io/name":                        "tempo-monolithic",
		"openshift.io/prometheus-rule-evaluation-scope": "leaf-prometheus",
		"monitoring": "enabled",
		"team":       "platform",
	}
	require.Equal(t, expectedLabels, rules.ObjectMeta.Labels)
}

package alerts

import (
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildPrometheusRule(t *testing.T) {
	params := manifestutils.Params{
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tempo-test",
				Namespace: "default",
			},
		},
	}
	objects, err := BuildPrometheusRule(params)

	require.NoError(t, err)
	assert.Len(t, objects, 1)
	rules := objects[0].(*monitoringv1.PrometheusRule)

	assert.Equal(t, "tempo-test-prometheus-rule", rules.Name)
}

func TestBuildPrometheusRuleWithExtraLabels(t *testing.T) {
	extraLabels := map[string]string{
		"monitoring": "enabled",
		"team":       "platform",
	}
	params := manifestutils.Params{
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tempo-test",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoStackSpec{
				Observability: v1alpha1.ObservabilitySpec{
					Metrics: v1alpha1.MetricsConfigSpec{
						CreatePrometheusRules:     true,
						ExtraPrometheusRuleLabels: extraLabels,
					},
				},
			},
		},
	}
	objects, err := BuildPrometheusRule(params)

	require.NoError(t, err)
	assert.Len(t, objects, 1)
	rules := objects[0].(*monitoringv1.PrometheusRule)

	expectedLabels := map[string]string{
		"app.kubernetes.io/instance":                    "tempo-test",
		"app.kubernetes.io/managed-by":                  "tempo-operator",
		"app.kubernetes.io/name":                        "tempo",
		"openshift.io/prometheus-rule-evaluation-scope": "leaf-prometheus",
		"monitoring": "enabled",
		"team":       "platform",
	}
	assert.Equal(t, expectedLabels, rules.ObjectMeta.Labels)
}

package status

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

var (
	metricTempoStackStatusCondition = promauto.With(metrics.Registry).NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempostack",
		Name:      "status_condition",
		Help:      "The status condition of a TempoStack instance.",
	}, []string{"stack_namespace", "stack_name", "condition"})
	metricTempoMonolithicStatusCondition = promauto.With(metrics.Registry).NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempomonolithic",
		Name:      "status_condition",
		Help:      "The status condition of a TempoMonolithic instance.",
	}, []string{"stack_namespace", "stack_name", "condition"})
)

func updateMetrics(metric *prometheus.GaugeVec, conditions []metav1.Condition, namespace string, name string) {
	// Update all status condition metrics.
	// In some cases not all status conditions are present in the status.Conditions list, for example:
	// A TempoStack CR gets created with an invalid storage secret (creating an ConfigurationError status condition).
	// Later this CR is deleted, a storage secret is created and a new TempoStack instance is created.
	// Then this TempoStack instance doesn't have the ConfigurationError condition in the status.Conditions list.
	activeConditions := map[string]float64{}
	for _, cond := range conditions {
		if cond.Status == metav1.ConditionTrue {
			activeConditions[cond.Type] = 1
		}
	}
	for _, cond := range v1alpha1.AllStatusConditions {
		condStr := string(cond)
		isActive := activeConditions[condStr] // isActive will be 0 if the condition is not found in the map
		metric.WithLabelValues(namespace, name, condStr).Set(isActive)
	}
}

// ClearTempoStackMetrics sets status condition metrics to zero.
func ClearTempoStackMetrics(namespace string, name string) {
	for _, cond := range v1alpha1.AllStatusConditions {
		condStr := string(cond)
		metricTempoStackStatusCondition.WithLabelValues(namespace, name, condStr).Set(0)
	}
}

// ClearMonolithicMetrics sets status condition metrics to zero.
func ClearMonolithicMetrics(namespace string, name string) {
	for _, cond := range v1alpha1.AllStatusConditions {
		condStr := string(cond)
		metricTempoMonolithicStatusCondition.WithLabelValues(namespace, name, condStr).Set(0)
	}
}

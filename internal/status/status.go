package status

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/version"
)

var (
	tempoStackStatusCondition = promauto.With(metrics.Registry).NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempostack",
		Name:      "status_condition",
		Help:      "The status condition of a TempoStack instance.",
	}, []string{"stack_namespace", "stack_name", "condition"})
)

// Refresh updates the status field with the tempo container image versions and updates the tempostack_status_condition metric.
func Refresh(ctx context.Context, k StatusClient, tempo v1alpha1.TempoStack, status *v1alpha1.TempoStackStatus) (bool, error) {
	changed := tempo.DeepCopy()
	changed.Status = *status

	// The .status.version field is empty for new CRs and cannot be set in the Defaulter webhook.
	// The upgrade procedure only runs once at operator startup, therefore we need to set
	// the initial status field versions here.
	if status.TempoVersion == "" {
		changed.Status.TempoVersion = version.Get().TempoVersion
	}
	if status.TempoQueryVersion == "" {
		changed.Status.TempoQueryVersion = version.Get().TempoQueryVersion
	}

	// Update all status condition metrics.
	// In some cases not all status conditions are present in the status.Conditions list, for example:
	// A TempoStack CR gets created with an invalid storage secret (creating a Degraded status condition).
	// Later this CR is deleted, a storage secret is created and a new TempoStack instance is created.
	// Then this TempoStack instance doesn't have the degraded condition in the status.Conditions list.
	activeConditions := map[string]float64{}
	for _, cond := range status.Conditions {
		if cond.Status == metav1.ConditionTrue {
			activeConditions[cond.Type] = 1
		}
	}
	for _, cond := range v1alpha1.AllStatusConditions {
		condStr := string(cond)
		isActive := activeConditions[condStr] // isActive will be 0 if the condition is not found in the map
		tempoStackStatusCondition.WithLabelValues(tempo.Namespace, tempo.Name, condStr).Set(isActive)
	}

	err := k.PatchStatus(ctx, changed, &tempo)
	if err != nil {
		return true, err
	}

	return false, nil
}

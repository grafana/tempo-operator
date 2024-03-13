package status

import (
	"context"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

// Refresh updates the status field with the Tempo versions and updates the tempostack_status_condition metric.
func Refresh(ctx context.Context, k StatusClient, tempo v1alpha1.TempoStack, status *v1alpha1.TempoStackStatus) error {
	changed := tempo.DeepCopy()
	changed.Status = *status

	updateMetrics(metricTempoStackStatusCondition, status.Conditions, tempo.Namespace, tempo.Name)

	err := k.PatchStatus(ctx, changed, &tempo)
	if err != nil {
		return err
	}

	return nil
}

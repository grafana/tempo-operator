package status

import (
	"context"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/version"
)

// Refresh updates the status field with the Tempo versions and updates the tempostack_status_condition metric.
func Refresh(ctx context.Context, k StatusClient, tempo v1alpha1.TempoStack, status *v1alpha1.TempoStackStatus) error {
	changed := tempo.DeepCopy()
	changed.Status = *status

	// The version fields in the status are empty for new CRs
	if status.OperatorVersion == "" {
		changed.Status.OperatorVersion = version.Get().OperatorVersion
	}
	if status.TempoVersion == "" {
		changed.Status.TempoVersion = version.Get().TempoVersion
	}

	updateMetrics(metricTempoStackStatusCondition, status.Conditions, tempo.Namespace, tempo.Name)

	err := k.PatchStatus(ctx, changed, &tempo)
	if err != nil {
		return err
	}

	return nil
}

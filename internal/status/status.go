package status

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/version"
)

// Refresh updates the status field with the Tempo versions and updates the tempostack_status_condition metric.
func Refresh(ctx context.Context, k StatusClient, tempo v1alpha1.TempoStack, status *v1alpha1.TempoStackStatus) error {
	// The version fields in the status are empty for new CRs
	if status.OperatorVersion == "" {
		status.OperatorVersion = version.Get().OperatorVersion
	}
	if status.TempoVersion == "" {
		status.TempoVersion = version.Get().TempoVersion
	}

	updateMetrics(metricTempoStackStatusCondition, status.Conditions, tempo.Namespace, tempo.Name)

	statusUpdater := func(tempo *v1alpha1.TempoStack) {
		tempo.Status = *status
	}

	statusUpdater(&tempo)
	// happy path: avoid extra k.Get()
	// Use Update instead of Patch: a strategic merge patch merges maps and cannot remove keys, but PodStatusMap needs to drop entries for removed pods.
	err := k.UpdateStatus(ctx, &tempo)
	if err == nil || !errors.IsConflict(err) {
		return err
	}

	// retry on conflict
	objectKey := client.ObjectKeyFromObject(&tempo)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := k.Get(ctx, objectKey, &tempo); err != nil {
			return err
		}

		statusUpdater(&tempo)
		return k.UpdateStatus(ctx, &tempo)
	})
}

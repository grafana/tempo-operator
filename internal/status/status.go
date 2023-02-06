package status

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

// Refresh executes an aggregate update of the Microservice Status struct, i.e.
// - It recreates the Status.Components pod status map per component.
// - It sets the appropriate Status.Condition to true that matches the pod status maps.
func Refresh(ctx context.Context, k StatusClient, s v1alpha1.Microservices) error {
	if err := SetComponentsStatus(ctx, k, s); err != nil {
		return err
	}

	cs := s.Status.Components

	// Check for failed pods first
	failed := len(cs.Compactor[corev1.PodFailed]) +
		len(cs.Distributor[corev1.PodFailed]) +
		len(cs.Ingester[corev1.PodFailed]) +
		len(cs.Querier[corev1.PodFailed]) +
		len(cs.QueryFrontend[corev1.PodFailed])

	unknown := len(cs.Compactor[corev1.PodUnknown]) +
		len(cs.Distributor[corev1.PodUnknown]) +
		len(cs.Ingester[corev1.PodUnknown]) +
		len(cs.Querier[corev1.PodUnknown]) +
		len(cs.QueryFrontend[corev1.PodUnknown])

	if failed != 0 || unknown != 0 {
		return SetFailedCondition(ctx, k, s)
	}

	// Check for pending pods
	pending := len(cs.Compactor[corev1.PodPending]) +
		len(cs.Distributor[corev1.PodPending]) +
		len(cs.Ingester[corev1.PodPending]) +
		len(cs.Querier[corev1.PodPending]) +
		len(cs.QueryFrontend[corev1.PodPending])

	if pending != 0 {
		return SetPendingCondition(ctx, k, s)
	}
	return SetReadyCondition(ctx, k, s)
}

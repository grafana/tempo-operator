package status

import (
	"context"

	"github.com/ViaQ/logerr/v2/kverrors"
	corev1 "k8s.io/api/core/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// SetComponentsStatus updates the pod status map component.
func componentsStatus(ctx context.Context, c StatusClient, s v1alpha1.TempoStack) (v1alpha1.ComponentStatus, error) {

	var err error
	components := v1alpha1.ComponentStatus{}
	components.Compactor, err = appendPodStatus(ctx, c, manifestutils.CompactorComponentName, s)
	if err != nil {
		return v1alpha1.ComponentStatus{}, kverrors.Wrap(err, "failed lookup TempoStack component pods status", "name", manifestutils.CompactorComponentName)
	}

	components.Querier, err = appendPodStatus(ctx, c, manifestutils.QuerierComponentName, s)
	if err != nil {
		return v1alpha1.ComponentStatus{}, kverrors.Wrap(err, "failed lookup TempoStack component pods status", "name", manifestutils.QuerierComponentName)
	}

	components.Distributor, err = appendPodStatus(ctx, c, manifestutils.DistributorComponentName, s)
	if err != nil {
		return v1alpha1.ComponentStatus{}, kverrors.Wrap(err, "failed lookup TempoStack component pods status", "name", manifestutils.DistributorComponentName)
	}

	components.QueryFrontend, err = appendPodStatus(ctx, c, manifestutils.QueryFrontendComponentName, s)
	if err != nil {
		return v1alpha1.ComponentStatus{}, kverrors.Wrap(err, "failed lookup TempoStack component pods status", "name", manifestutils.QueryFrontendComponentName)
	}

	components.Ingester, err = appendPodStatus(ctx, c, manifestutils.IngesterComponentName, s)
	if err != nil {
		return v1alpha1.ComponentStatus{}, kverrors.Wrap(err, "failed lookup TempoStack component pods status", "name", manifestutils.IngesterComponentName)
	}

	components.Gateway, err = appendPodStatus(ctx, c, manifestutils.GatewayComponentName, s)
	if err != nil {
		return v1alpha1.ComponentStatus{}, kverrors.Wrap(err, "failed lookup TempoStack component pods status", "name", manifestutils.GatewayComponentName)
	}

	return components, nil
}

func appendPodStatus(ctx context.Context, c StatusClient, componentName string, stack v1alpha1.TempoStack) (v1alpha1.PodStatusMap, error) {
	psm := v1alpha1.PodStatusMap{}
	pods, err := c.GetPodsComponent(ctx, componentName, stack)

	if err != nil {
		return nil, kverrors.Wrap(err, "failed to list pods for TempoStack component", "name", stack, "component", componentName)
	}

	for _, pod := range pods.Items {
		phase := pod.Status.Phase
		psm[phase] = append(psm[phase], pod.Name)
	}
	return psm, nil
}

// GetComponentsStatus executes an aggregate update of the TempoStack Status struct, i.e.
// - It recreates the Status.Components pod status map per component.
// - It sets the appropriate Status.Condition to true that matches the pod status maps.
func GetComponentsStatus(ctx context.Context, k StatusClient, s v1alpha1.TempoStack) (v1alpha1.TempoStackStatus, error) {

	cs, err := componentsStatus(ctx, k, s)
	if err != nil {
		return v1alpha1.TempoStackStatus{}, err
	}
	s.Status.Components = cs

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
		s.Status.Conditions = FailedCondition(s)
		return s.Status, nil
	}

	// Check for pending pods
	pending := len(cs.Compactor[corev1.PodPending]) +
		len(cs.Distributor[corev1.PodPending]) +
		len(cs.Ingester[corev1.PodPending]) +
		len(cs.Querier[corev1.PodPending]) +
		len(cs.QueryFrontend[corev1.PodPending])

	if pending != 0 {
		s.Status.Conditions = PendingCondition(s)
		return s.Status, nil

	}
	s.Status.Conditions = ReadyCondition(s)
	return s.Status, nil
}

package status

import (
	"context"

	"github.com/ViaQ/logerr/v2/kverrors"
	corev1 "k8s.io/api/core/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func podStatus(pod *corev1.Pod) v1alpha1.PodStatus {
	switch pod.Status.Phase {
	case corev1.PodFailed:
		return v1alpha1.PodFailed
	case corev1.PodPending:
		return v1alpha1.PodPending
	case corev1.PodSucceeded:
		return v1alpha1.PodSucceeded
	case corev1.PodUnknown:
		return v1alpha1.PodStatusUnknown
	case corev1.PodRunning:
		for _, c := range pod.Status.ContainerStatuses {
			if !c.Ready {
				return v1alpha1.PodRunning
			}
		}
		return v1alpha1.PodReady
	default:
		return v1alpha1.PodStatusUnknown
	}
}

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

	if s.Spec.Template.MetricsGenerator.Enabled {
		components.MetricsGenerator, err = appendPodStatus(ctx, c, manifestutils.MetricsGeneratorComponentName, s)
		if err != nil {
			return v1alpha1.ComponentStatus{}, kverrors.Wrap(err, "failed lookup TempoStack component pods status", "name", manifestutils.MetricsGeneratorComponentName)
		}
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
		if pod.GetDeletionTimestamp() != nil {
			continue
		}

		status := podStatus(&pod)
		psm[status] = append(psm[status], pod.Name)
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
	failed := len(cs.Compactor[v1alpha1.PodFailed]) +
		len(cs.Distributor[v1alpha1.PodFailed]) +
		len(cs.Ingester[v1alpha1.PodFailed]) +
		len(cs.Querier[v1alpha1.PodFailed]) +
		len(cs.QueryFrontend[v1alpha1.PodFailed]) +
		len(cs.MetricsGenerator[v1alpha1.PodFailed])

	unknown := len(cs.Compactor[v1alpha1.PodStatusUnknown]) +
		len(cs.Distributor[v1alpha1.PodStatusUnknown]) +
		len(cs.Ingester[v1alpha1.PodStatusUnknown]) +
		len(cs.Querier[v1alpha1.PodStatusUnknown]) +
		len(cs.QueryFrontend[v1alpha1.PodStatusUnknown]) +
		len(cs.MetricsGenerator[v1alpha1.PodStatusUnknown])

	if failed != 0 || unknown != 0 {
		s.Status.Conditions = FailedCondition(s)
		return s.Status, nil
	}

	// Check for pending pods
	pending := len(cs.Compactor[v1alpha1.PodPending]) +
		len(cs.Distributor[v1alpha1.PodPending]) +
		len(cs.Ingester[v1alpha1.PodPending]) +
		len(cs.Querier[v1alpha1.PodPending]) +
		len(cs.QueryFrontend[v1alpha1.PodPending]) +
		len(cs.MetricsGenerator[v1alpha1.PodPending])

	// Check for running (not ready) pods
	running := len(cs.Compactor[v1alpha1.PodRunning]) +
		len(cs.Distributor[v1alpha1.PodRunning]) +
		len(cs.Ingester[v1alpha1.PodRunning]) +
		len(cs.Querier[v1alpha1.PodRunning]) +
		len(cs.QueryFrontend[v1alpha1.PodRunning]) +
		len(cs.MetricsGenerator[v1alpha1.PodRunning])

	if pending != 0 || running != 0 {
		s.Status.Conditions = PendingCondition(s)
		return s.Status, nil

	}
	s.Status.Conditions = ReadyCondition(s)
	return s.Status, nil
}

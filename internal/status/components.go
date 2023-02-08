package status

import (
	"context"

	"github.com/ViaQ/logerr/v2/kverrors"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
)

// SetComponentsStatus updates the pod status map component.
func SetComponentsStatus(ctx context.Context, c StatusClient, s v1alpha1.Microservices) error {

	var err error
	original := s.DeepCopy()
	s.Status.Components = v1alpha1.ComponentStatus{}
	s.Status.Components.Compactor, err = appendPodStatus(ctx, c, manifestutils.CompactorComponentName, s)
	if err != nil {
		return kverrors.Wrap(err, "failed lookup Microservice component pods status", "name", manifestutils.CompactorComponentName)
	}

	s.Status.Components.Querier, err = appendPodStatus(ctx, c, manifestutils.QuerierComponentName, s)
	if err != nil {
		return kverrors.Wrap(err, "failed lookup Microservice component pods status", "name", manifestutils.QuerierComponentName)
	}

	s.Status.Components.Distributor, err = appendPodStatus(ctx, c, manifestutils.DistributorComponentName, s)
	if err != nil {
		return kverrors.Wrap(err, "failed lookup Microservice component pods status", "name", manifestutils.DistributorComponentName)
	}

	s.Status.Components.QueryFrontend, err = appendPodStatus(ctx, c, manifestutils.QueryFrontendComponentName, s)
	if err != nil {
		return kverrors.Wrap(err, "failed lookup Microservice component pods status", "name", manifestutils.QueryFrontendComponentName)
	}

	s.Status.Components.Ingester, err = appendPodStatus(ctx, c, manifestutils.IngesterComponentName, s)
	if err != nil {
		return kverrors.Wrap(err, "failed lookup Microservice component pods status", "name", manifestutils.IngesterComponentName)
	}

	return c.PatchStatus(ctx, &s, original)
}

func appendPodStatus(ctx context.Context, c StatusClient, componentName string, stack v1alpha1.Microservices) (v1alpha1.PodStatusMap, error) {
	psm := v1alpha1.PodStatusMap{}
	pods, err := c.GetPodsComponent(ctx, componentName, stack)

	if err != nil {
		return nil, kverrors.Wrap(err, "failed to list pods for Microservice component", "name", stack, "component", componentName)
	}

	for _, pod := range pods.Items {
		phase := pod.Status.Phase
		psm[phase] = append(psm[phase], pod.Name)
	}
	return psm, nil
}

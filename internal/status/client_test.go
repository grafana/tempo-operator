package status

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

type statusClientStub struct {
	GetPodsComponentStub func(ctx context.Context, componentName string, stack v1alpha1.TempoStack) (*corev1.PodList, error)
	UpdateStatusStub     func(ctx context.Context, s v1alpha1.TempoStack) error
	PatchStatusStub      func(ctx context.Context, changed, original *v1alpha1.TempoStack) error
}

func (scs *statusClientStub) GetPodsComponent(ctx context.Context, componentName string, stack v1alpha1.TempoStack) (*corev1.PodList, error) {
	return scs.GetPodsComponentStub(ctx, componentName, stack)
}

func (scs *statusClientStub) PatchStatus(ctx context.Context, changed, original *v1alpha1.TempoStack) error {
	return scs.PatchStatusStub(ctx, changed, original)
}

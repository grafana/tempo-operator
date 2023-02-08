package status

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

type StatusClient interface {
	GetPodsComponent(ctx context.Context, componentName string, stack v1alpha1.Microservices) (*corev1.PodList, error)
	PatchStatus(ctx context.Context, changed, original *v1alpha1.Microservices) error
}

type StatusClientStub struct {
	GetPodsComponentStub func(ctx context.Context, componentName string, stack v1alpha1.Microservices) (*corev1.PodList, error)
	UpdateStatusStub     func(ctx context.Context, s v1alpha1.Microservices) error
	PatchStatusStub      func(ctx context.Context, changed, original *v1alpha1.Microservices) error
}

func (scs *StatusClientStub) GetPodsComponent(ctx context.Context, componentName string, stack v1alpha1.Microservices) (*corev1.PodList, error) {
	return scs.GetPodsComponentStub(ctx, componentName, stack)
}

func (scs *StatusClientStub) PatchStatus(ctx context.Context, changed, original *v1alpha1.Microservices) error {
	return scs.PatchStatusStub(ctx, changed, original)
}

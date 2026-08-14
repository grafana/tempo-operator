package status

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

type statusClientStub struct {
	GetStub              func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error
	GetPodsComponentStub func(ctx context.Context, componentName string, stack v1alpha1.TempoStack) (*corev1.PodList, error)
	UpdateStatusStub     func(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error
}

func (scs *statusClientStub) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if scs.GetStub != nil {
		return scs.GetStub(ctx, key, obj, opts...)
	}
	return nil
}

func (scs *statusClientStub) GetPodsComponent(ctx context.Context, componentName string, stack v1alpha1.TempoStack) (*corev1.PodList, error) {
	return scs.GetPodsComponentStub(ctx, componentName, stack)
}

func (scs *statusClientStub) UpdateStatus(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	return scs.UpdateStatusStub(ctx, obj, opts...)
}

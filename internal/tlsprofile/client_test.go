package tlsprofile

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type clientStub struct {
	GetStub func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error
}

func (scs *clientStub) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return scs.GetStub(ctx, key, obj, opts...)
}

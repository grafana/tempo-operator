package tlsprofile

import (
	"context"

	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type clientStub struct {
	mock.Mock
}

func (scs2 *clientStub) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	args := scs2.Called(ctx, key, obj, opts)
	return args.Error(0)
}

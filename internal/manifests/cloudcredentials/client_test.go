package cloudcredentials

import (
	"context"

	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type clientStub struct {
	mock.Mock
}

func (c *clientStub) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	args := c.Called(ctx, key, obj, opts)
	return args.Error(0)
}

func (c *clientStub) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	args := c.Called(ctx, obj, opts)
	return args.Error(0)

}
func (c *clientStub) CreateOrUpdate(ctx context.Context, obj client.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	args := c.Called(ctx, obj, f)
	return args.Get(0).(controllerutil.OperationResult), args.Error(1)
}

package tlsprofile

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type k8getter interface {
	Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error
}

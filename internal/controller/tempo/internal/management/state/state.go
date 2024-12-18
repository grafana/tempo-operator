package state

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"

	"github.com/ViaQ/logerr/v2/kverrors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

// IsManaged checks if the custom resource is configured with ManagementState Managed.
func IsManaged(ctx context.Context, req ctrl.Request, k client.Client) (bool, error) {
	var stack tempov1alpha1.TempoStack
	if err := k.Get(ctx, req.NamespacedName, &stack); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, kverrors.Wrap(err, "failed to lookup tempostack", "name", req.NamespacedName)
	}
	return stack.Spec.ManagementState == tempov1alpha1.ManagementStateManaged, nil
}

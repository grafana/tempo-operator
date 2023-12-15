package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/internal/manifests"
)

func isNamespaceScoped(obj client.Object) bool {
	switch obj.(type) {
	case *rbacv1.ClusterRole, *rbacv1.ClusterRoleBinding:
		return false
	default:
		return true
	}
}

// reconcileManagedObjects creates or updates all managed objects.
// If immutable fields are changed, the object will be deleted and re-created.
func reconcileManagedObjects(ctx context.Context, log logr.Logger, k8sclient client.Client, owner metav1.Object, scheme *runtime.Scheme, managedObjects []client.Object) error {
	errs := []error{}
	for _, obj := range managedObjects {
		l := log.WithValues(
			"objectName", obj.GetName(),
			"objectKind", obj.GetObjectKind().GroupVersionKind(),
		)

		if isNamespaceScoped(obj) {
			if err := ctrl.SetControllerReference(owner, obj, scheme); err != nil {
				l.Error(err, "failed to set controller owner reference to resource")
				errs = append(errs, err)
				continue
			}
		}

		desired := obj.DeepCopyObject().(client.Object)
		mutateFn := manifests.MutateFuncFor(obj, desired)

		op, err := ctrl.CreateOrUpdate(ctx, k8sclient, obj, mutateFn)

		var immutableErr *manifests.ImmutableErr
		if err != nil && errors.As(err, &immutableErr) {
			l.Error(err, "detected a change in an immutable field. The object will be deleted, and re-created on next reconcile", "obj", obj.GetName())
			err = k8sclient.Delete(ctx, desired)
		}
		if err != nil {
			l.Error(err, "failed to configure resource")
			errs = append(errs, err)
			continue
		}

		l.V(1).Info(fmt.Sprintf("resource has been %s", op))
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to create objects for %s: %w", owner.GetName(), errors.Join(errs...))
	}
	return nil
}

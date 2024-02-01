package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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
func reconcileManagedObjects(
	ctx context.Context,
	log logr.Logger,
	k8sclient client.Client,
	owner metav1.Object,
	scheme *runtime.Scheme,
	managedObjects []client.Object,
	ownedObjects map[types.UID]client.Object,
) error {
	pruneObjects := ownedObjects

	// Create or update all objects managed by the operator
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

		var op controllerutil.OperationResult
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			var err error
			op, err = ctrl.CreateOrUpdate(ctx, k8sclient, obj, mutateFn)
			return err
		})

		var immutableErr *manifests.ImmutableErr
		if err != nil && errors.As(err, &immutableErr) {
			l.Error(err, "detected a change in an immutable field. The object will be deleted, and re-created on next reconcile", "obj", obj.GetName())
			err = k8sclient.Delete(ctx, desired)
		}

		if err != nil {
			l.Error(err, "failed to configure resource")
			errs = append(errs, err)
		} else {
			l.V(1).Info(fmt.Sprintf("resource has been %s", op))
		}

		// This object is still managed by the operator, remove it from the list of objects to prune
		delete(pruneObjects, obj.GetUID())
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to create objects for %s: %w", owner.GetName(), errors.Join(errs...))
	}

	// Prune owned objects in the cluster which are not managed anymore
	pruneErrs := []error{}
	for _, obj := range pruneObjects {
		l := log.WithValues(
			"objectName", obj.GetName(),
			"objectKind", obj.GetObjectKind(),
		)

		l.Info("pruning unmanaged resource")
		err := k8sclient.Delete(ctx, obj)
		if err != nil {
			l.Error(err, "failed to delete resource")
			pruneErrs = append(pruneErrs, err)
		}
	}
	if len(pruneErrs) > 0 {
		return fmt.Errorf("failed to prune objects for %s: %w", owner.GetName(), errors.Join(pruneErrs...))
	}

	return nil
}

// listFieldErrors converts field.ErrorList to a comma separated string of errors.
func listFieldErrors(fieldErrs field.ErrorList) string {
	msgs := make([]string, len(fieldErrs))
	for i, fieldErr := range fieldErrs {
		msgs[i] = fieldErr.Detail
	}
	return strings.Join(msgs, ", ")
}

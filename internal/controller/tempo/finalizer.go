package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func finalize(ctx context.Context, cl client.Client, logger logr.Logger, objectLabels map[string]string) error {
	// The cluster scope objects do not have owner reference. They need to be deleted explicitly
	objects, err := findClusterRoleObjects(ctx, cl, objectLabels)
	if err != nil {
		return err
	}
	return deleteObjects(ctx, cl, logger, objects)
}

func deleteObjects(ctx context.Context, kubeClient client.Client, logger logr.Logger, objects map[types.UID]client.Object) error {
	// Pruning owned objects in the cluster which are not should not be present after the reconciliation.
	pruneErrs := []error{}
	for _, obj := range objects {
		l := logger.WithValues(
			"object_name", obj.GetName(),
			"object_kind", obj.GetObjectKind().GroupVersionKind(),
		)

		l.Info("pruning unmanaged resource")
		err := kubeClient.Delete(ctx, obj)
		if err != nil {
			l.Error(err, "failed to delete resource")
			pruneErrs = append(pruneErrs, err)
		}
	}
	return errors.Join(pruneErrs...)
}

// The cluster scope objects do not have owner reference.
func findClusterRoleObjects(ctx context.Context, cl client.Client, objectLabels map[string]string) (map[types.UID]client.Object, error) {
	ownedObjects := map[types.UID]client.Object{}
	listOpsCluster := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(objectLabels),
	}

	for _, objectType := range []client.Object{&rbacv1.ClusterRole{}, &rbacv1.ClusterRoleBinding{}} {
		objs, err := getList(ctx, cl, objectType, listOpsCluster)
		if err != nil {
			return nil, err
		}
		for uid, object := range objs {
			ownedObjects[uid] = object
		}
	}
	return ownedObjects, nil
}

// getList queries the Kubernetes API to list the requested resource, setting the list l of type T.
func getList[T client.Object](ctx context.Context, cl client.Client, l T, options ...client.ListOption) (map[types.UID]client.Object, error) {
	ownedObjects := map[types.UID]client.Object{}
	gvk, err := apiutil.GVKForObject(l, cl.Scheme())
	if err != nil {
		return nil, err
	}
	gvk.Kind = fmt.Sprintf("%sList", gvk.Kind)
	list, err := cl.Scheme().New(gvk)
	if err != nil {
		return nil, fmt.Errorf("unable to list objects of type %s: %w", gvk.Kind, err)
	}

	objList := list.(client.ObjectList)

	err = cl.List(ctx, objList, options...)
	if err != nil {
		return ownedObjects, fmt.Errorf("error listing %T: %w", l, err)
	}
	objs, err := apimeta.ExtractList(objList)
	if err != nil {
		return ownedObjects, fmt.Errorf("error listing %T: %w", l, err)
	}
	for i := range objs {
		typedObj, ok := objs[i].(T)
		if !ok {
			return ownedObjects, fmt.Errorf("error listing %T: %w", l, err)
		}
		ownedObjects[typedObj.GetUID()] = typedObj
	}
	return ownedObjects, nil
}

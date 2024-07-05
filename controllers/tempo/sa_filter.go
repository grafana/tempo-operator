package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// Filter service account objects already created and modified by OCP. e.g  bound service account tokens
// when generating pull secrets adds an annotation to the SA. In such case we are not interested on modified it.
func filterServiceAccountObjects(ctx context.Context,
	cl client.Client, tempo metav1.ObjectMeta, objects []client.Object) ([]client.Object, error) {

	var filtered []client.Object

	serviceAccountList := &corev1.ServiceAccountList{}
	err := cl.List(ctx, serviceAccountList,
		&client.ListOptions{
			Namespace:     tempo.GetNamespace(),
			LabelSelector: labels.SelectorFromSet(manifestutils.CommonLabels(tempo.Name)),
		},
	)

	if err != nil {
		return nil, err
	}

	for _, o := range objects {
		switch newSA := o.(type) {
		case *corev1.ServiceAccount:
			needsUpdate := true
			for _, existingSA := range serviceAccountList.Items {
				if existingSA.Name == newSA.Name {
					// may be is not enough to verify for existence, we need to fine tune this part
					needsUpdate = false
				}
			}
			if needsUpdate {
				filtered = append(filtered, o)
			}
		default:
			filtered = append(filtered, o)
		}
	}

	return filtered, nil
}

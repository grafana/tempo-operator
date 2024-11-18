package storage

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

var (
	// ErrFetchingConfigMap is used in the webhook to not fail validation when there was an error retrieving the ConfigMap.
	// Manifests can be applied out-of-order (i.e. the CR gets applied before the ConfigMap).
	ErrFetchingConfigMap = "could not fetch ConfigMap"
)

func getTLSParams(ctx context.Context, client client.Client, namespace string, tlsSpec v1alpha1.TLSSpec, configMapPath *field.Path) (manifestutils.StorageTLS, field.ErrorList) {
	tlsParams := manifestutils.StorageTLS{}

	if tlsSpec.CA != "" {
		caConfigMap, errs := getCAConfigMap(ctx, client, namespace, tlsSpec.CA, configMapPath)
		if len(errs) > 0 {
			return manifestutils.StorageTLS{}, errs
		}

		tlsParams.CAFilename, errs = getCAConfigMapKey(caConfigMap, configMapPath)
		if len(errs) > 0 {
			return manifestutils.StorageTLS{}, errs
		}
	}

	return tlsParams, nil
}

func getCAConfigMap(ctx context.Context, client client.Client, namespace string, name string, path *field.Path) (corev1.ConfigMap, field.ErrorList) {
	var caConfigMap corev1.ConfigMap
	err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &caConfigMap)
	if err != nil {
		return corev1.ConfigMap{}, field.ErrorList{field.Invalid(path, name, fmt.Sprintf("%s: %v", ErrFetchingConfigMap, err))}
	}

	return caConfigMap, nil
}

func getCAConfigMapKey(caConfigMap corev1.ConfigMap, path *field.Path) (string, field.ErrorList) {
	if caConfigMap.Data[manifestutils.TLSCAFilename] != "" {
		return manifestutils.TLSCAFilename, nil
	}

	// for backwards compatibility
	if caConfigMap.Data[manifestutils.StorageTLSCAFilename] != "" {
		return manifestutils.StorageTLSCAFilename, nil
	}

	return "", field.ErrorList{field.Invalid(path, caConfigMap.Name, fmt.Sprintf("CA ConfigMap must contain a '%s' key", manifestutils.TLSCAFilename))}
}

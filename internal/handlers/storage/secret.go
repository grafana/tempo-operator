package storage

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// ErrFetchingSecret is used in the webhook to not fail validation when there was an error retrieving the storage secret.
	// Manifests can be applied out-of-order (i.e. the CR gets applied before the storage secret).
	ErrFetchingSecret = "could not fetch Secret"
	hashSeparator     = []byte(",")
)

func getSecret(ctx context.Context, client client.Client, namespace string, secretName string, path *field.Path) (corev1.Secret, field.ErrorList) {
	var storageSecret corev1.Secret
	err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: secretName}, &storageSecret)
	if err != nil {
		return corev1.Secret{}, field.ErrorList{field.Invalid(path, secretName, fmt.Sprintf("%s: %v", ErrFetchingSecret, err))}
	}

	return storageSecret, nil
}

func ensureNotEmpty(storageSecret corev1.Secret, fields []string, path *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	for _, key := range fields {
		if storageSecret.Data[key] == nil || len(storageSecret.Data[key]) == 0 {
			allErrs = append(allErrs, field.Invalid(
				path,
				storageSecret.Name,
				fmt.Sprintf("storage secret must contain \"%s\" field", key),
			))
		}
	}
	return allErrs
}

func hashSecretData(s *corev1.Secret) (string, error) {
	keys := make([]string, 0, len(s.Data))
	for k := range s.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := sha256.New()
	for _, k := range keys {
		if _, err := h.Write([]byte(k)); err != nil {
			return "", err
		}

		if _, err := h.Write(hashSeparator); err != nil {
			return "", err
		}

		if _, err := h.Write(s.Data[k]); err != nil {
			return "", err
		}

		if _, err := h.Write(hashSeparator); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

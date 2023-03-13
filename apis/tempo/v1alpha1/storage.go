package v1alpha1

import (
	"fmt"
	"net/url"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateStorageSecret validates the object storage secret required for tempo.
func ValidateStorageSecret(tempo TempoStack, storageSecret corev1.Secret) field.ErrorList {
	path := field.NewPath("spec").Child("storage").Child("secret")

	if storageSecret.Data == nil {
		return field.ErrorList{field.Invalid(path, tempo.Spec.Storage.Secret, "storage secret is empty")}
	}

	var allErrs field.ErrorList

	switch tempo.Spec.Storage.Secret.Type {
	case ObjectStorageSecretAzure:
		allErrs = append(allErrs, validateAzureSecret(tempo, path, storageSecret)...)
	case ObjectStorageSecretS3:
		allErrs = append(allErrs, validateS3Secret(tempo, path, storageSecret)...)
	case "":
		allErrs = append(allErrs, field.Invalid(
			path,
			tempo.Spec.Storage.Secret,
			"storage secret must specify the type",
		))
	default:
		allErrs = append(allErrs, field.Invalid(
			path,
			tempo.Spec.Storage.Secret,
			fmt.Sprintf("%s is not an allowed storage secret type", tempo.Spec.Storage.Secret.Type),
		))
	}

	return allErrs
}

func ensureNotEmpty(tempo TempoStack, path *field.Path, storageSecret corev1.Secret, fields []string) field.ErrorList {
	var allErrs field.ErrorList
	for _, key := range fields {
		if storageSecret.Data[key] == nil || len(storageSecret.Data[key]) == 0 {
			allErrs = append(allErrs, field.Invalid(
				path,
				tempo.Spec.Storage.Secret,
				fmt.Sprintf("storage secret must contain \"%s\" field", key),
			))
		}
	}
	return allErrs
}

func validateAzureSecret(tempo TempoStack, path *field.Path, storageSecret corev1.Secret) field.ErrorList {
	var allErrs field.ErrorList
	secretFields := []string{
		"container",
		"account_name",
		"account_key",
	}

	allErrs = append(allErrs, ensureNotEmpty(tempo, path, storageSecret, secretFields)...)
	return allErrs
}

func validateS3Secret(tempo TempoStack, path *field.Path, storageSecret corev1.Secret) field.ErrorList {
	var allErrs field.ErrorList
	secretFields := []string{
		"endpoint",
		"bucket",
		"access_key_id",
		"access_key_secret",
	}

	allErrs = append(allErrs, ensureNotEmpty(tempo, path, storageSecret, secretFields)...)

	if endpoint, ok := storageSecret.Data["endpoint"]; ok {
		u, err := url.ParseRequestURI(string(endpoint))

		// ParseRequestURI also accepts absolute paths, therefore we need to check if the URL scheme is set
		if err != nil || u.Scheme == "" {
			allErrs = append(allErrs, field.Invalid(
				path,
				tempo.Spec.Storage.Secret,
				"\"endpoint\" field of storage secret must be a valid URL",
			))
		}
	}

	return allErrs
}

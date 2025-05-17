package storage

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

var (
	azureSecretFields = []string{
		"container",
		"account_name",
		"account_key",
	}
)

func discoverAzureCredentialType(_ corev1.Secret, _ *field.Path) (v1alpha1.CredentialMode, field.ErrorList) {
	// Currently the only mode supported by Azure.
	return v1alpha1.CredentialModeStatic, nil
}

func validateAzureSecret(storageSecret corev1.Secret, path *field.Path, _ v1alpha1.CredentialMode) field.ErrorList {
	return ensureNotEmpty(storageSecret, azureSecretFields, path)
}

func getAzureParams(storageSecret corev1.Secret, path *field.Path, mode v1alpha1.CredentialMode) (*manifestutils.AzureStorage, field.ErrorList) {
	errs := validateAzureSecret(storageSecret, path, mode)
	if len(errs) != 0 {
		return nil, errs
	}

	return &manifestutils.AzureStorage{
		Container: string(storageSecret.Data["container"]),
	}, nil
}

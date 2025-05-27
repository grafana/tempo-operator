package storage

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

var (
	azureLongLivedFields = []string{
		"container",
		"account_name",
		"account_key",
	}

	azureShortLivedFields = []string{
		"container",
		"account_name",
		"client_id",
		"tenant_id",
	}
)

func discoverAzureCredentialType(secret corev1.Secret, path *field.Path) (v1alpha1.CredentialMode, field.ErrorList) {
	var isShortLived bool
	_, ok := secret.Data["tenant_id"]
	if ok {
		isShortLived = true
	}

	var isLongLived bool
	_, ok = secret.Data["account_key"]
	if ok {
		isLongLived = true
	}

	if isShortLived && isLongLived {
		return "", field.ErrorList{field.Invalid(
			path,
			secret.Name,
			"storage secret contains fields for long lived and short lived configuration",
		)}
	}
	if isShortLived {
		return v1alpha1.CredentialModeToken, nil
	}
	return v1alpha1.CredentialModeStatic, nil
}

func validateAzureSecret(storageSecret corev1.Secret, path *field.Path, credentialMode v1alpha1.CredentialMode) field.ErrorList {
	switch credentialMode {
	case v1alpha1.CredentialModeStatic:
		return ensureNotEmpty(storageSecret, azureLongLivedFields, path)
	case v1alpha1.CredentialModeToken:
		return ensureNotEmpty(storageSecret, azureShortLivedFields, path)
	case v1alpha1.CredentialModeTokenCCO:
		return field.ErrorList{field.Invalid(
			path,
			credentialMode,
			"not supported by azure secret credential mode",
		)}
	}
	return field.ErrorList{}
}

func getAzureParams(storageSecret corev1.Secret, path *field.Path, mode v1alpha1.CredentialMode) (*manifestutils.AzureStorage, field.ErrorList) {
	errs := validateAzureSecret(storageSecret, path, mode)
	if len(errs) != 0 {
		return nil, errs
	}

	if mode == v1alpha1.CredentialModeStatic {
		return &manifestutils.AzureStorage{
			Container: string(storageSecret.Data["container"]),
		}, nil
	}

	audience := ""
	if mode == v1alpha1.CredentialModeToken {
		if aud, ok := storageSecret.Data["audience"]; ok {
			audience = string(aud)
		} else {
			audience = manifestutils.AzureDefaultAudience
		}
	}

	return &manifestutils.AzureStorage{
		Container: string(storageSecret.Data["container"]),
		ClientID:  string(storageSecret.Data["client_id"]),
		TenantID:  string(storageSecret.Data["tenant_id"]),
		Audience:  audience,
	}, nil

}

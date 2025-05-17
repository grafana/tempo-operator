package storage

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

var (
	gcsShortLivedFields = []string{
		"bucketname",
		"iam_sa",
		"iam_sa_project_id",
	}

	gcsLongLivedFields = []string{
		"bucketname",
		"key.json",
	}
)

func discoverGCSCredentialType(storageSecret corev1.Secret, path *field.Path) (v1alpha1.CredentialMode, field.ErrorList) {
	// ship bucketname as it is common for both
	var isShortLived bool
	for _, v := range gcsShortLivedFields[1:] {
		_, ok := storageSecret.Data[v]
		if ok {
			isShortLived = true
		}
	}
	var isLongLived bool
	for _, v := range gcsLongLivedFields[1:] {
		_, ok := storageSecret.Data[v]
		if ok {
			isLongLived = true
		}
	}

	if isShortLived && isLongLived {
		return "", field.ErrorList{field.Invalid(
			path,
			storageSecret.Name,
			"storage secret contains fields for long lived and short lived configuration",
		)}
	}

	if isShortLived {
		return v1alpha1.CredentialModeToken, nil
	}

	return v1alpha1.CredentialModeStatic, nil
}

func validateGCSSecret(storageSecret corev1.Secret, path *field.Path, credentialMode v1alpha1.CredentialMode) field.ErrorList {
	switch credentialMode {
	case v1alpha1.CredentialModeStatic:
		return ensureNotEmpty(storageSecret, gcsLongLivedFields, path)
	case v1alpha1.CredentialModeToken:
		return ensureNotEmpty(storageSecret, gcsShortLivedFields, path)
	case v1alpha1.CredentialModeTokenCCO:
		return ensureNotEmpty(storageSecret, gcsShortLivedFields, path)
	}
	return field.ErrorList{}
}

func getGCSParams(storageSecret corev1.Secret, path *field.Path, mode v1alpha1.CredentialMode) (*manifestutils.GCS, field.ErrorList) {

	errs := validateGCSSecret(storageSecret, path, mode)
	if len(errs) != 0 {
		return nil, errs
	}

	if mode == v1alpha1.CredentialModeToken {
		return &manifestutils.GCS{
			Bucket:            string(storageSecret.Data["bucketname"]),
			IAMServiceAccount: string(storageSecret.Data["iam_sa"]),
			ProjectID:         string(storageSecret.Data["iam_sa_project_id"]),
		}, nil
	}

	return &manifestutils.GCS{
		Bucket: string(storageSecret.Data["bucketname"]),
	}, nil
}

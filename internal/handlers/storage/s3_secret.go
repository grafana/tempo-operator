package storage

import (
	"net/url"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

var s3ShortLivedFields = []string{
	"bucket",
	"region",
	"role_arn",
}

var s3CCOShortLivedFields = []string{
	"bucket",
	"region",
}

var s3LongLivedFields = []string{
	"bucket",
	"endpoint",
	"access_key_id",
	"access_key_secret",
}

func discoverS3CredentialType(storageSecret corev1.Secret, path *field.Path) (v1alpha1.CredentialMode, field.ErrorList) {

	var isShortLived bool
	for _, v := range s3ShortLivedFields[1:] {
		_, ok := storageSecret.Data[v]
		if ok {
			isShortLived = true
		}
	}
	var isLongLived bool
	for _, v := range s3LongLivedFields[1:] {
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

func validateS3Secret(storageSecret corev1.Secret, path *field.Path, credentialMode v1alpha1.CredentialMode) field.ErrorList {
	switch credentialMode {
	case v1alpha1.CredentialModeStatic:
		var allErrs field.ErrorList
		allErrs = append(allErrs, ensureNotEmpty(storageSecret, s3LongLivedFields, path)...)
		if endpoint, ok := storageSecret.Data["endpoint"]; ok {
			u, err := url.ParseRequestURI(string(endpoint))

			// ParseRequestURI also accepts absolute paths, therefore we need to check if the URL scheme is set
			if err != nil || u.Scheme == "" {
				allErrs = append(allErrs, field.Invalid(
					path,
					storageSecret.Name,
					"\"endpoint\" field of storage secret must be a valid URL",
				))
			}
		}
		return allErrs
	case v1alpha1.CredentialModeToken:
		return ensureNotEmpty(storageSecret, s3ShortLivedFields, path)
	case v1alpha1.CredentialModeTokenCCO:
		return ensureNotEmpty(storageSecret, s3CCOShortLivedFields, path)
	}

	return field.ErrorList{}
}

func getS3Params(storageSecret corev1.Secret, path *field.Path, mode v1alpha1.CredentialMode) (*manifestutils.S3, field.ErrorList) {

	errs := validateS3Secret(storageSecret, path, mode)
	if len(errs) != 0 {
		return nil, errs
	}

	if mode == v1alpha1.CredentialModeStatic {
		endpoint := string(storageSecret.Data["endpoint"])
		insecure := !strings.HasPrefix(endpoint, "https://")
		endpoint = strings.TrimPrefix(endpoint, "https://")
		endpoint = strings.TrimPrefix(endpoint, "http://")
		return &manifestutils.S3{
			Insecure: insecure,
			Endpoint: endpoint,
			Bucket:   string(storageSecret.Data["bucket"]),
		}, nil
	}

	if mode == v1alpha1.CredentialModeToken {
		return &manifestutils.S3{
			Bucket:  string(storageSecret.Data["bucket"]),
			RoleARN: string(storageSecret.Data["role_arn"]),
			Region:  string(storageSecret.Data["region"]),
		}, nil
	}

	return &manifestutils.S3{
		Bucket: string(storageSecret.Data["bucket"]),
		Region: string(storageSecret.Data["region"]),
	}, nil
}

package storage

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

var (
	// ErrFetchingSecret is used in the webhook to not fail validation when there was an error retrieving the storage secret.
	// Manifests can be applied out-of-order (i.e. the CR gets applied before the storage secret).
	ErrFetchingSecret = "could not fetch Secret"
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

func validateS3Secret(storageSecret corev1.Secret, path *field.Path) field.ErrorList {
	shortLivedFields := []string{
		"bucket",
		"region",
		"role_arn",
	}
	longLivedFields := []string{
		"bucket",
		"endpoint",
		"access_key_id",
		"access_key_secret",
	}

	// ship bucket as it is common for both
	var isShortLived bool
	for _, v := range shortLivedFields[1:] {
		_, ok := storageSecret.Data[v]
		if ok {
			isShortLived = true
		}
	}
	var isLongLived bool
	for _, v := range longLivedFields[1:] {
		_, ok := storageSecret.Data[v]
		if ok {
			isLongLived = true
		}
	}

	if isShortLived && isLongLived {
		return field.ErrorList{field.Invalid(
			path,
			storageSecret.Name,
			"storage secret contains fields for long lived and short lived configuration",
		)}
	}

	// check short-lived first
	if storageSecret.Data["role_arn"] != nil || storageSecret.Data["region"] != nil {
		return ensureNotEmpty(storageSecret, shortLivedFields, path)
	}

	var allErrs field.ErrorList
	allErrs = append(allErrs, ensureNotEmpty(storageSecret, longLivedFields, path)...)

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
}

// getS3Params extracts S3 params from the storage secret.
func getS3Params(storageSecret corev1.Secret, path *field.Path) (*manifestutils.S3, field.ErrorList) {
	errs := validateS3Secret(storageSecret, path)
	if len(errs) > 0 {
		return nil, errs
	}

	if storageSecret.Data["role_arn"] != nil || storageSecret.Data["region"] != nil {
		return &manifestutils.S3{
			ShortLived: &manifestutils.S3ShortLived{
				Bucket:  string(storageSecret.Data["bucket"]),
				RoleARN: string(storageSecret.Data["role_arn"]),
				Region:  string(storageSecret.Data["region"]),
			},
		}, nil
	}

	endpoint := string(storageSecret.Data["endpoint"])
	insecure := !strings.HasPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	return &manifestutils.S3{
		Insecure: insecure,
		LongLived: &manifestutils.S3LongLived{
			Endpoint: endpoint,
			Bucket:   string(storageSecret.Data["bucket"]),
		},
	}, nil
}

func validateAzureSecret(storageSecret corev1.Secret, path *field.Path) field.ErrorList {
	secretFields := []string{
		"container",
		"account_name",
		"account_key",
	}
	return ensureNotEmpty(storageSecret, secretFields, path)
}

// getAzureParams extracts Azure Storage params from the storage secret.
func getAzureParams(storageSecret corev1.Secret, path *field.Path) (*manifestutils.AzureStorage, field.ErrorList) {
	errs := validateAzureSecret(storageSecret, path)
	if len(errs) > 0 {
		return nil, errs
	}

	return &manifestutils.AzureStorage{
		Container: string(storageSecret.Data["container"]),
	}, nil
}

func validateGCSSecret(storageSecret corev1.Secret, path *field.Path) field.ErrorList {
	shortLivedFields := []string{
		"bucketname",
		"iam_sa",
		"iam_sa_project_id",
	}

	longLivedFields := []string{
		"bucketname",
		"key.json",
	}

	// ship bucketname as it is common for both
	var isShortLived bool
	for _, v := range shortLivedFields[1:] {
		_, ok := storageSecret.Data[v]
		if ok {
			isShortLived = true
		}
	}
	var isLongLived bool
	for _, v := range longLivedFields[1:] {
		_, ok := storageSecret.Data[v]
		if ok {
			isLongLived = true
		}
	}

	if isShortLived && isLongLived {
		return field.ErrorList{field.Invalid(
			path,
			storageSecret.Name,
			"storage secret contains fields for long lived and short lived configuration",
		)}
	}

	// check short-lived first
	if storageSecret.Data["iam_sa"] != nil || storageSecret.Data["iam_sa_project_id"] != nil {
		return ensureNotEmpty(storageSecret, shortLivedFields, path)
	}

	var allErrs field.ErrorList
	allErrs = append(allErrs, ensureNotEmpty(storageSecret, longLivedFields, path)...)

	return allErrs
}

// getGCSParams extracts GCS params from the storage secret.
func getGCSParams(storageSecret corev1.Secret, path *field.Path) (*manifestutils.GCS, field.ErrorList) {
	errs := validateGCSSecret(storageSecret, path)
	if len(errs) > 0 {
		return nil, errs
	}

	if storageSecret.Data["iam_sa"] != nil && storageSecret.Data["iam_sa_project_id"] != nil {
		return &manifestutils.GCS{
			Bucket: string(storageSecret.Data["bucketname"]),
			ShortLived: &manifestutils.GCSShortLived{
				IAMServiceAccount: string(storageSecret.Data["iam_sa"]),
				ProjectID:         string(storageSecret.Data["iam_sa_project_id"]),
			},
		}, nil
	}

	return &manifestutils.GCS{
		Bucket: string(storageSecret.Data["bucketname"]),
	}, nil
}

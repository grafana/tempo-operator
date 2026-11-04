package storage

import (
	"fmt"
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

// s3ShortLivedDiscoveryFields are the fields which identify a secret holding short lived
// credentials. "bucket" is not part of it, because it is common to both credential types.
var s3ShortLivedDiscoveryFields = []string{
	"region",
	"role_arn",
}

// s3LongLivedDiscoveryFields are the fields which identify a secret holding long lived
// credentials. Neither "bucket" nor "endpoint" are part of it, because both are common
// to long lived credentials and to short lived credentials outside of the commercial
// AWS partition.
var s3LongLivedDiscoveryFields = []string{
	"access_key_id",
	"access_key_secret",
}

func discoverS3CredentialType(storageSecret corev1.Secret, path *field.Path) (v1alpha1.CredentialMode, field.ErrorList) {

	var isShortLived bool
	for _, v := range s3ShortLivedDiscoveryFields {
		_, ok := storageSecret.Data[v]
		if ok {
			isShortLived = true
		}
	}
	var isLongLived bool
	for _, v := range s3LongLivedDiscoveryFields {
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
		allErrs = append(allErrs, validateS3Endpoint(storageSecret, path)...)
		return allErrs
	case v1alpha1.CredentialModeToken:
		var allErrs field.ErrorList
		allErrs = append(allErrs, ensureNotEmpty(storageSecret, s3ShortLivedFields, path)...)
		allErrs = append(allErrs, validateS3Endpoint(storageSecret, path)...)
		return allErrs
	case v1alpha1.CredentialModeTokenCCO:
		var allErrs field.ErrorList
		allErrs = append(allErrs, ensureNotEmpty(storageSecret, s3CCOShortLivedFields, path)...)
		allErrs = append(allErrs, validateS3Endpoint(storageSecret, path)...)
		return allErrs
	}

	return field.ErrorList{}
}

func getS3Params(storageSecret corev1.Secret, path *field.Path, mode v1alpha1.CredentialMode) (*manifestutils.S3, field.ErrorList) {
	errs := validateS3Secret(storageSecret, path, mode)
	if len(errs) != 0 {
		return nil, errs
	}

	switch mode {
	case v1alpha1.CredentialModeStatic:
		endpoint, insecure := parseS3Endpoint(string(storageSecret.Data["endpoint"]))
		return &manifestutils.S3{
			Insecure: insecure,
			Endpoint: endpoint,
			Bucket:   string(storageSecret.Data["bucket"]),
		}, nil

	case v1alpha1.CredentialModeToken, v1alpha1.CredentialModeTokenCCO:
		return getS3ShortLivedParams(storageSecret, mode), nil

	default:
		return nil, field.ErrorList{field.Invalid(
			path,
			mode,
			fmt.Sprintf("unsupported credential mode %q", mode),
		)}
	}
}

// getS3ShortLivedParams returns the S3 parameters of the token and token-cco credential modes.
func getS3ShortLivedParams(storageSecret corev1.Secret, mode v1alpha1.CredentialMode) *manifestutils.S3 {
	region := string(storageSecret.Data["region"])
	partition := manifestutils.AWSPartitionForRegion(region)

	// The endpoint is optional for short lived credentials: by default it is derived from the
	// region and the partition it belongs to. An explicit endpoint supports private endpoints,
	// and its scheme is the only way to reach S3 over plain HTTP in these modes.
	endpoint := partition.S3Endpoint(region)
	insecure := false
	if rawEndpoint, ok := storageSecret.Data["endpoint"]; ok {
		endpoint, insecure = parseS3Endpoint(string(rawEndpoint))
	}

	// The audience must match a client ID of the IAM OIDC identity provider of the cluster.
	// ccoctl registers sts.amazonaws.com in every partition, and it is the default of the
	// pod identity webhook, therefore it is not derived from the partition.
	audience := manifestutils.AWSDefaultAudience
	if rawAudience, ok := storageSecret.Data["audience"]; ok {
		audience = string(rawAudience)
	}

	s3 := &manifestutils.S3{
		Bucket:    string(storageSecret.Data["bucket"]),
		Region:    region,
		Endpoint:  endpoint,
		Insecure:  insecure,
		Partition: partition,
		Audience:  audience,
	}

	if mode == v1alpha1.CredentialModeToken {
		s3.RoleARN = string(storageSecret.Data["role_arn"])
	}

	return s3
}

// validateS3Endpoint ensures the optional "endpoint" field of a storage secret is a valid URL.
func validateS3Endpoint(storageSecret corev1.Secret, path *field.Path) field.ErrorList {
	endpoint, ok := storageSecret.Data["endpoint"]
	if !ok {
		return nil
	}

	u, err := url.ParseRequestURI(string(endpoint))

	// ParseRequestURI also accepts absolute paths, therefore we need to check if the URL scheme is set
	if err != nil || u.Scheme == "" {
		return field.ErrorList{field.Invalid(
			path,
			storageSecret.Name,
			"\"endpoint\" field of storage secret must be a valid URL",
		)}
	}

	return nil
}

// parseS3Endpoint splits an endpoint URL into the host[:port] expected by Tempo and the
// insecure flag derived from its scheme.
func parseS3Endpoint(rawEndpoint string) (string, bool) {
	insecure := !strings.HasPrefix(rawEndpoint, "https://")
	endpoint := strings.TrimPrefix(rawEndpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	return endpoint, insecure
}

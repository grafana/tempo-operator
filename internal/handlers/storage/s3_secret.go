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

	if mode == v1alpha1.CredentialModeStatic {
		endpoint, insecure := parseS3Endpoint(string(storageSecret.Data["endpoint"]))
		return &manifestutils.S3{
			Insecure: insecure,
			Endpoint: endpoint,
			Bucket:   string(storageSecret.Data["bucket"]),
		}, nil
	}

	region := string(storageSecret.Data["region"])
	partition := manifestutils.AWSPartitionForRegion(region)

	// The endpoint is optional for short lived credentials: by default it is derived from the
	// region and the partition it belongs to. An explicit endpoint supports private endpoints,
	// and is the only way to reach S3 over plain HTTP, which is required in the ISO partitions,
	// where service control policies block HTTPS access to S3.
	endpoint := partition.S3Endpoint(region)
	insecure := false
	if rawEndpoint, ok := storageSecret.Data["endpoint"]; ok {
		endpoint, insecure = parseS3Endpoint(string(rawEndpoint))
	}

	audience := manifestutils.AWSDefaultAudience
	if !partition.IsCommercial() {
		audience = partition.STSEndpoint(region)
	}
	if rawAudience, ok := storageSecret.Data["audience"]; ok {
		audience = string(rawAudience)
	}

	// Tempo obtains its AWS credentials through the credential chain of the minio-go client,
	// which resolves the STS endpoint to sts.<region>.amazonaws.com. Partitions serving their
	// endpoints under a different DNS suffix must override it, see ConfigureS3Storage.
	stsEndpoint := ""
	if !partition.IsCommercial() {
		stsEndpoint = "https://" + partition.STSEndpoint(region)
	}

	s3 := &manifestutils.S3{
		Bucket:      string(storageSecret.Data["bucket"]),
		Region:      region,
		Endpoint:    endpoint,
		Insecure:    insecure,
		Partition:   partition.ID,
		Audience:    audience,
		STSEndpoint: stsEndpoint,
	}

	if mode == v1alpha1.CredentialModeToken {
		s3.RoleARN = string(storageSecret.Data["role_arn"])
	}

	return s3, nil
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

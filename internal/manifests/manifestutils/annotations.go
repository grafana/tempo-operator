package manifestutils

import (
	"crypto/sha256"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// CommonAnnotations returns common annotations for each pod created by the operator.
func CommonAnnotations(configChecksum string) map[string]string {
	return map[string]string{
		"tempo.grafana.com/config.hash": configChecksum,
	}
}

// S3AWSSTSAnnotations returns service account annotations required by AWS STS.
func S3AWSSTSAnnotations(secret S3) map[string]string {
	return map[string]string{
		"eks.amazonaws.com/audience": "sts.amazonaws.com",
		"eks.amazonaws.com/role-arn": secret.RoleARN,
	}
}

// AzureShortLiveTokenAnnotation returns service account annotations required by Azure Short Live Token.
func AzureShortLiveTokenAnnotation(secret AzureStorage) map[string]string {
	return map[string]string{
		"azure.workload.identity/client-id": secret.ClientID,
		"azure.workload.identity/tenant-id": secret.TenantID,
	}
}

// StorageSecretHash return annotations for secret storage content hashes.
func StorageSecretHash(params StorageParams, annotations map[string]string) map[string]string {
	if params.CloudCredentials.ContentHash != "" {
		annotations["tempo.grafana.com/token.cco.auth.hash"] = params.CloudCredentials.ContentHash
	}

	return annotations
}

// AddCertificateHashAnnotations adds restart-related annotations from TempoStack CR to trigger pod restarts.
func AddCertificateHashAnnotations(crAnnotations map[string]string, annotations map[string]string) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Copy certificate hash annotations from CR to pod templates to trigger rolling updates
	const certHashPrefix = "tempo.grafana.com/cert-hash-"
	for key, value := range crAnnotations {
		if strings.HasPrefix(key, certHashPrefix) {
			annotations[key] = value
		}
	}

	return annotations
}

// CertificateHashAnnotations calculates and returns certificate hash annotations from certificate secrets.
func CertificateHashAnnotations(certSecrets map[string]*corev1.Secret) map[string]string {
	annotations := make(map[string]string)

	for name, secret := range certSecrets {
		if secret != nil && secret.Data != nil {
			// Calculate hash of the certificate data
			if certData, exists := secret.Data[corev1.TLSCertKey]; exists {
				hash := sha256.Sum256(certData)
				annotations[fmt.Sprintf("tempo.grafana.com/cert-hash-%s", name)] = fmt.Sprintf("%x", hash)
			}
		}
	}

	return annotations
}

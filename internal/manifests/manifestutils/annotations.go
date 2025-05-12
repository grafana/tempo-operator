package manifestutils

import "fmt"

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

// GCSShortLiveTokenAnnotation returns service account annotations required by GCS Short Live Token.
func GCSShortLiveTokenAnnotation(secret GCS) map[string]string {
	return map[string]string{
		"iam.gke.io/gcp-service-account": fmt.Sprintf("%s@%s.iam.gserviceaccount.com",
			secret.IAMServiceAccount, secret.ProjectID),
	}
}

// AzureShortLiveTokenAnnotation returns service account annotations required by GCS Short Live Token.
func AzureShortLiveTokenAnnotation(secret AzureStorage) map[string]string {
	return map[string]string{
		"azure.workload.identity/client-id":  secret.ClientID,
		"azure.workload.identity/tenantt-id": secret.TenantID,
	}
}

// StorageSecretHash return annotations for secret storage content hashes.
func StorageSecretHash(params StorageParams, annotations map[string]string) map[string]string {
	if params.CloudCredentials.ContentHash != "" {
		annotations["tempo.grafana.com/token.cco.auth.hash"] = params.CloudCredentials.ContentHash
	}

	return annotations
}

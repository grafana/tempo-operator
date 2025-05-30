package manifestutils

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

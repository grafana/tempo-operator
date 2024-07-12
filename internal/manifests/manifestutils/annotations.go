package manifestutils

// CommonAnnotations returns common annotations for each pod created by the operator.
func CommonAnnotations(configChecksum string) map[string]string {
	return map[string]string{
		"tempo.grafana.com/config.hash": configChecksum,
	}
}

// S3AWSSTSAnnotations returns service account annotations required by AWS STS.
func S3AWSSTSAnnotations(secret S3ShortLived) map[string]string {
	return map[string]string{
		"eks.amazonaws.com/audience": "sts.amazonaws.com",
		"eks.amazonaws.com/role-arn": secret.RoleARN,
	}

}

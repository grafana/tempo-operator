package manifestutils

import "fmt"

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

// GCSShortLiveTokenAnnotation returns service account annotations required by GCS Short Live Token.
func GCSShortLiveTokenAnnotation(secret GCSShortLived) map[string]string {
	return map[string]string{
		"iam.gke.io/gcp-service-account": fmt.Sprintf("%s@%s.iam.gserviceaccount.com",
			secret.IAMServiceAccount, secret.ProjectID),
	}
}

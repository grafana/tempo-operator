package controllers

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// GetAzureParams extracts Azure storage params of a storage secret.
func GetAzureParams(storageSecret *corev1.Secret) *manifestutils.AzureStorage {
	return &manifestutils.AzureStorage{
		Container: string(storageSecret.Data["container"]),
	}
}

// GetGCSParams extracts GCS params of a storage secret.
func GetGCSParams(storageSecret *corev1.Secret) *manifestutils.GCS {
	return &manifestutils.GCS{
		Bucket: string(storageSecret.Data["bucketname"]),
	}
}

// GetS3Params extracts S3 params of a storage secret.
func GetS3Params(storageSecret *corev1.Secret) *manifestutils.S3 {
	endpoint := string(storageSecret.Data["endpoint"])
	insecure := !strings.HasPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	return &manifestutils.S3{
		Endpoint: endpoint,
		Bucket:   string(storageSecret.Data["bucket"]),
		Insecure: insecure,
	}
}

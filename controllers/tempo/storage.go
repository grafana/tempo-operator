package controllers

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func getAzureParams(storageSecret *corev1.Secret) *manifestutils.AzureStorage {
	return &manifestutils.AzureStorage{
		Container: string(storageSecret.Data["container"]),
	}
}

func getGCSParams(storageSecret *corev1.Secret) *manifestutils.GCS {
	return &manifestutils.GCS{
		Bucket: string(storageSecret.Data["bucketname"]),
	}
}

func getS3Params(storageSecret *corev1.Secret) *manifestutils.S3 {
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

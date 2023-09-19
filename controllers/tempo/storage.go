package controllers

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// GetAzureParams extracts Azure Storage params from the storage secret.
func GetAzureParams(tempo v1alpha1.TempoStack, storageSecret *corev1.Secret) *manifestutils.AzureStorage {
	return &manifestutils.AzureStorage{
		Container: string(storageSecret.Data["container"]),
	}
}

// GetGCSParams extracts GCS params from the storage secret.
func GetGCSParams(tempo v1alpha1.TempoStack, storageSecret *corev1.Secret) *manifestutils.GCS {
	return &manifestutils.GCS{
		Bucket: string(storageSecret.Data["bucketname"]),
	}
}

// GetS3Params extracts S3 params from the storage secret.
func GetS3Params(tempo v1alpha1.TempoStack, storageSecret *corev1.Secret) *manifestutils.S3 {
	endpoint := string(storageSecret.Data["endpoint"])
	insecure := !strings.HasPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	caPath := ""
	if tempo.Spec.Storage.TLS.CA != "" {
		caPath = manifestutils.TempoStorageTLSCAPath()
	}

	return &manifestutils.S3{
		Endpoint:  endpoint,
		Bucket:    string(storageSecret.Data["bucket"]),
		Insecure:  insecure,
		TLSCAPath: caPath,
	}
}

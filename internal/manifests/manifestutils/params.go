package manifestutils

import (
	configv1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/tlsprofile"
)

// Params holds parameters used to create Tempo objects.
type Params struct {
	StorageParams  StorageParams
	ConfigChecksum string
	Tempo          v1alpha1.TempoStack
	Gates          configv1alpha1.FeatureGates
	TLSProfile     tlsprofile.TLSProfileOptions
}

// StorageParams holds storage configuration.
type StorageParams struct {
	AzureStorage *AzureStorage
	GCS          *GCS
	S3           *S3
}

// AzureStorage for Azure Storage.
type AzureStorage struct {
	Container   string
	AccountName string
	AccountKey  string
}

// GCS for Google Cloud Storage.
type GCS struct {
	Bucket  string
	KeyJson string
}

// S3 holds S3 configuration.
type S3 struct {
	Endpoint string
	Bucket   string
}

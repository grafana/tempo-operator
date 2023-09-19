package manifestutils

import (
	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/tlsprofile"
)

// Params holds parameters used to create Tempo objects.
type Params struct {
	StorageParams       StorageParams
	ConfigChecksum      string
	Tempo               v1alpha1.TempoStack
	Gates               configv1alpha1.FeatureGates
	TLSProfile          tlsprofile.TLSProfileOptions
	GatewayTenantSecret []*GatewayTenantOIDCSecret
	GatewayTenantsData  []*GatewayTenantsData
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
	// Endpoint without http/https
	Endpoint  string
	Bucket    string
	Insecure  bool
	TLSCAPath string
}

// GatewayTenantOIDCSecret holds clientID, clientSecret and issuerCAPath for tenant's authentication.
type GatewayTenantOIDCSecret struct {
	TenantName   string
	ClientID     string
	ClientSecret string
	IssuerCAPath string
}

// GatewayTenantsData holds cookie secret for opa-openshift sidecar.
type GatewayTenantsData struct {
	TenantName string
	// OpenShiftCookieSecret is used for encrypting the auth token when put into the browser session.
	OpenShiftCookieSecret string
}

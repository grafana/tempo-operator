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
	CtrlConfig          configv1alpha1.ProjectConfig
	TLSProfile          tlsprofile.TLSProfileOptions
	GatewayTenantSecret []*GatewayTenantOIDCSecret
	GatewayTenantsData  []*GatewayTenantsData
}

// StorageParams holds storage configuration from the storage secret, except the credentials.
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
	Bucket string
}

// S3 holds S3 configuration.
type S3 struct {
	LongLived  *S3LongLived
	ShortLived *S3ShortLived
	Insecure   bool
}

// S3LongLived holds long-lived S3 configuration.
// The long-lived token uses access key and secret.
type S3LongLived struct {
	// Endpoint without http/https
	Endpoint string
	Bucket   string
	TLS      StorageTLS
}

// S3ShortLived holds short-lived S3 configuration.
// The short-lived S3 token uses AWS STS.
type S3ShortLived struct {
	Bucket  string
	RoleARN string
	Region  string
}

// StorageTLS holds StorageTLS configuration.
type StorageTLS struct {
	CAFilename string // for backwards compatibility (can be service-ca.crt or ca.crt)
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

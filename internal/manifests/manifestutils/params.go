package manifestutils

import (
	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
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
	AzureStorage     *AzureStorage
	GCS              *GCS
	S3               *S3
	CredentialMode   v1alpha1.CredentialMode
	CloudCredentials CloudCredentials
}

// CloudCredentials secret details.
type CloudCredentials struct {
	ContentHash string
	Environment *TokenCCOAuthConfig
}

// AzureStorage for Azure Storage.
type AzureStorage struct {
	Container  string
	AccountKey string
	ClientID   string
	TenantID   string
	Audience   string
}

// GCS for Google Cloud Storage.
type GCS struct {
	Bucket            string
	IAMServiceAccount string
	ProjectID         string
	Audience          string
}

// S3 holds S3 configuration.
type S3 struct {
	Endpoint string
	TLS      StorageTLS
	Bucket   string
	RoleARN  string
	Region   string
	Insecure bool
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

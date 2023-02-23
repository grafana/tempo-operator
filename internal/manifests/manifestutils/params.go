package manifestutils

import (
	"strings"

	configv1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

// Params holds parameters used to create Tempo objects.
type Params struct {
	StorageParams  StorageParams
	ConfigChecksum string
	Tempo          v1alpha1.Microservices
	Gates          configv1alpha1.FeatureGates
	TLSProfile     TLSProfileOptions
}

// StorageParams holds storage configuration.
type StorageParams struct {
	S3 S3
}

// S3 holds S3 configuration.
type S3 struct {
	Endpoint string
	Bucket   string
}

// TLSProfileOptions is the desired behavior of a TLSProfileType.
type TLSProfileOptions struct {
	// Ciphers is used to specify the cipher algorithms that are negotiated
	// during the TLS handshake.
	Ciphers []string
	// MinTLSVersion is used to specify the minimal version of the TLS protocol
	// that is negotiated during the TLS handshake.
	MinTLSVersion string
}

// TLSCipherSuites transforms TLSProfileSpec.Ciphers from a slice
// to a string of elements joined with a comma.
func (o TLSProfileOptions) TLSCipherSuites() string {
	return strings.Join(o.Ciphers, ",")
}

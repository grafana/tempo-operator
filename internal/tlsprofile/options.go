package tlsprofile

import (
	"errors"
	"strings"

	openshiftconfigv1 "github.com/openshift/api/config/v1"
)

// ErrInvalidTLSVersion is returned when the TLS version is invalid.
var ErrInvalidTLSVersion = errors.New("invalid TLS version")

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

// MinVersionShort returns the min TLS version but only the number instead of VersionTLS10 it will return 1.0.
func (o TLSProfileOptions) MinVersionShort() (string, error) {
	switch o.MinTLSVersion {
	case string(openshiftconfigv1.VersionTLS10):
		return "1.0", nil
	case string(openshiftconfigv1.VersionTLS11):
		return "1.1", nil
	case string(openshiftconfigv1.VersionTLS12):
		return "1.2", nil
	case string(openshiftconfigv1.VersionTLS13):
		return "1.3", nil
	default:
		return "", ErrInvalidTLSVersion
	}
}

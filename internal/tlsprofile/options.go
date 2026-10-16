package tlsprofile

import (
	"crypto/tls"
	"errors"
	"strings"

	openshiftconfigv1 "github.com/openshift/api/config/v1"
)

// ErrInvalidTLSVersion is returned when the TLS version is invalid.
var ErrInvalidTLSVersion = errors.New("invalid TLS version")

// TLSProfileOptions is the desired behavior of a TLSProfileType.
type TLSProfileOptions struct {
	// MinTLSVersion is used to specify the minimal version of the TLS protocol
	// that is negotiated during the TLS handshake.
	MinTLSVersion string
	// Ciphers is used to specify the cipher algorithms that are negotiated
	// during the TLS handshake.
	Ciphers []string
	// Groups contains IANA TLS Supported Groups names for key exchange.
	Groups []string
}

// MinVersionOTELFormat returns the min TLS version but only the number instead of VersionTLS10 it will return 1.0.
func (o TLSProfileOptions) MinVersionOTELFormat() string {
	switch o.MinTLSVersion {
	case string(openshiftconfigv1.VersionTLS10):
		return "1.0"
	case string(openshiftconfigv1.VersionTLS11):
		return "1.1"
	case string(openshiftconfigv1.VersionTLS12):
		return "1.2"
	case string(openshiftconfigv1.VersionTLS13):
		return "1.3"
	default:
		return ""
	}
}

// CipherSuites transforms TLSProfileSpec.Ciphers from a slice
// to a string of elements joined with a comma.
func (o TLSProfileOptions) CipherSuites() string {
	return strings.Join(o.Ciphers, ",")
}

// CurvePreferencesGateway returns the a comma-separated list of IANA group names.
//
// Supported curves:
// https://github.com/observatorium/api/blob/c73556f/tls/config.go
func (o TLSProfileOptions) CurvePreferencesGateway() string {
	return strings.Join(o.Groups, ",")
}

// CurvePreferencesDSKit converts IANA group names to dskit server TLS curve names.
//
// Supported curves:
// https://github.com/grafana/tempo/blob/v2.10.8/go.mod
// https://github.com/JoaoBraveCoding/dskit/blob/32ff95159224/server/tls_config.go
func (o TLSProfileOptions) CurvePreferencesDSKit() string {
	var curves []string
	for _, group := range o.Groups {
		switch openshiftconfigv1.TLSGroup(group) {
		case openshiftconfigv1.TLSGroupSecP256r1:
			curves = append(curves, tls.CurveP256.String())
		case openshiftconfigv1.TLSGroupSecP384r1:
			curves = append(curves, tls.CurveP384.String())
		case openshiftconfigv1.TLSGroupSecP521r1:
			curves = append(curves, tls.CurveP521.String())
		case openshiftconfigv1.TLSGroupX25519,
			openshiftconfigv1.TLSGroupX25519MLKEM768,
			openshiftconfigv1.TLSGroupSecP256r1MLKEM768,
			openshiftconfigv1.TLSGroupSecP384r1MLKEM1024:
			curves = append(curves, group)
		}
	}
	return strings.Join(curves, ",")
}

// CurvePreferencesOTEL converts IANA group names to OpenTelemetry TLS curve names.
//
// Supported curves:
// https://github.com/grafana/tempo/blob/v2.10.8/go.mod
// https://github.com/open-telemetry/opentelemetry-collector/blob/config/configtls/v1.56.0/config/configtls/curves_nofips.go#L10-L16
func (o TLSProfileOptions) CurvePreferencesOTEL() []string {
	var curves []string
	for _, group := range o.Groups {
		switch openshiftconfigv1.TLSGroup(group) {
		case openshiftconfigv1.TLSGroupSecP256r1:
			curves = append(curves, "P256")
		case openshiftconfigv1.TLSGroupSecP384r1:
			curves = append(curves, "P384")
		case openshiftconfigv1.TLSGroupSecP521r1:
			curves = append(curves, "P521")
		case openshiftconfigv1.TLSGroupX25519, openshiftconfigv1.TLSGroupX25519MLKEM768:
			curves = append(curves, group)
		case openshiftconfigv1.TLSGroupSecP256r1MLKEM768,
			openshiftconfigv1.TLSGroupSecP384r1MLKEM1024:
			continue
		}
	}
	return curves
}

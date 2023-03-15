package tlsprofile

import "strings"

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

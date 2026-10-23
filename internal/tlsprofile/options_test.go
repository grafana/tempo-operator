package tlsprofile

import (
	"testing"

	openshiftconfigv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
)

func TestOptionsCiphers(t *testing.T) {
	ops := TLSProfileOptions{
		Ciphers:       []string{"CIPHER1", "CIPHER2", "CIPHER3"},
		MinTLSVersion: "TLSv1.1",
	}

	expected := "CIPHER1,CIPHER2,CIPHER3"
	assert.Equal(t, expected, ops.CipherSuites())
}

func TestCurvePreferencesDSKit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		groups []string
		want   string
	}{
		{name: "unset"},
		{name: "empty", groups: []string{}},
		{
			name:   "classical groups preserve order",
			groups: []string{"secp521r1", "X25519", "secp256r1", "secp384r1"},
			want:   "CurveP521,X25519,CurveP256,CurveP384",
		},
		{
			name:   "hybrid groups preserve order with classical groups",
			groups: []string{"X25519MLKEM768", "X25519", "SecP256r1MLKEM768", "secp256r1", "SecP384r1MLKEM1024"},
			want:   "X25519MLKEM768,X25519,SecP256r1MLKEM768,CurveP256,SecP384r1MLKEM1024",
		},
		{name: "only unsupported groups", groups: []string{"unknown"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := TLSProfileOptions{Groups: tc.groups}
			assert.Equal(t, tc.want, opts.CurvePreferencesDSKit())
		})
	}
}

func TestCurvePreferencesOTEL(t *testing.T) {
	for _, tc := range []struct {
		name   string
		groups []string
		want   []string
	}{
		{name: "unset"},
		{name: "empty", groups: []string{}},
		{
			name:   "supported groups preserve order",
			groups: []string{"X25519MLKEM768", "secp521r1", "X25519", "secp256r1", "secp384r1"},
			want:   []string{"X25519MLKEM768", "P521", "X25519", "P256", "P384"},
		},
		{
			name:   "unsupported groups are omitted",
			groups: []string{"SecP256r1MLKEM768", "secp256r1", "SecP384r1MLKEM1024", "unknown"},
			want:   []string{"P256"},
		},
		{name: "only unsupported groups", groups: []string{"unknown"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := TLSProfileOptions{Groups: tc.groups}
			assert.Equal(t, tc.want, opts.CurvePreferencesOTEL())
		})
	}
}

func TestShortVersion(t *testing.T) {
	type testCase struct {
		version  openshiftconfigv1.TLSProtocolVersion
		expected string
	}
	for _, scenario := range []testCase{
		{
			version:  openshiftconfigv1.VersionTLS10,
			expected: "1.0",
		},
		{
			version:  openshiftconfigv1.VersionTLS11,
			expected: "1.1",
		},
		{
			version:  openshiftconfigv1.VersionTLS12,
			expected: "1.2",
		},
		{
			version:  openshiftconfigv1.VersionTLS13,
			expected: "1.3",
		},
		{
			version:  "invalid",
			expected: "",
		},
	} {
		t.Run(string(scenario.version), func(t *testing.T) {
			ops := TLSProfileOptions{
				MinTLSVersion: string(scenario.version),
			}
			v := ops.MinVersionOTELFormat()
			assert.Equal(t, scenario.expected, v)
		})
	}

}

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

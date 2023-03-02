package tlsprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptionsCiphers(t *testing.T) {
	ops := TLSProfileOptions{
		Ciphers:       []string{"CIPHER1", "CIPHER2", "CIPHER3"},
		MinTLSVersion: "TLSv1.1",
	}

	expected := "CIPHER1,CIPHER2,CIPHER3"
	assert.Equal(t, expected, ops.TLSCipherSuites())
}

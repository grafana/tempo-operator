package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFallbackVersion(t *testing.T) {
	assert.Equal(t, "0.0.0", Tempo())
}

func TestVersionFromBuild(t *testing.T) {
	// prepare
	tempo = "0.0.2" // set during the build
	defer func() {
		tempo = ""
	}()

	assert.Equal(t, tempo, Tempo())
	assert.Contains(t, Get().String(), tempo)
}

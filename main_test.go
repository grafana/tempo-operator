package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func TestSetupLogging(t *testing.T) {
	prevStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	setupLogging()
	log := log.FromContext(context.Background())
	log = log.WithValues("tempo", "simplest")
	log.Error(errors.New("test error"), "a test error occurred")

	err := w.Close()
	require.NoError(t, err)
	output, _ := io.ReadAll(r)
	os.Stderr = prevStderr

	require.Regexp(t, fmt.Sprintf(`%d.+ERROR\s+a test error occurred\s+{"tempo": "simplest", "error": "test error"}`, time.Now().Year()), string(output))
}

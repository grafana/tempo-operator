package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/log"

	configv1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
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

func TestReadConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected configv1alpha1.ProjectConfig
		err      string
	}{
		{
			name:  "no featureGates.tlsProfile given, using default value",
			input: "testdata/empty.yaml",
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					TLSProfile: string(configv1alpha1.TLSProfileModernType),
				},
			},
		},
		{
			name:  "featureGates.tlsProfile given, not using default value",
			input: "testdata/tlsprofile_old.yaml",
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					TLSProfile: string(configv1alpha1.TLSProfileOldType),
				},
			},
		},
		{
			name:  "invalid featureGates.tlsProfile given, show error",
			input: "testdata/tlsprofile_invalid.yaml",
			err:   "controller config validation failed: invalid value 'abc' for setting featureGates.tlsProfile (valid values: Old, Intermediate and Modern)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())

			err := readConfig(cmd, test.input)
			if test.err == "" {
				require.NoError(t, err)

				rootCmdConfig := cmd.Context().Value(RootConfigKey{}).(RootConfig)
				assert.Equal(t, test.expected, rootCmdConfig.CtrlConfig)
			} else {
				require.Error(t, err)
				require.Equal(t, test.err, err.Error())
			}
		})
	}
}

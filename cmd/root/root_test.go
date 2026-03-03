package root

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
)

func TestReadConfig(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		modifyCheck func(t *testing.T, cfg configv1alpha1.ProjectConfig)
		err         string
	}{
		{
			name:  "no featureGates.tlsProfile given, using default value",
			input: "../testdata/empty.yaml",
			modifyCheck: func(t *testing.T, cfg configv1alpha1.ProjectConfig) {
				// Config file doesn't override TLSProfile, so default "Modern" is used
				assert.Equal(t, configv1alpha1.TLSProfileModernType, cfg.Gates.TLSProfile)
				// Verify other community defaults are present
				assert.Equal(t, "community", cfg.Distribution)
				assert.True(t, cfg.Gates.HTTPEncryption)
				assert.True(t, cfg.Gates.GRPCEncryption)
				assert.True(t, cfg.Gates.NetworkPolicies)
				assert.True(t, cfg.Gates.BuiltInCertManagement.Enabled)
			},
		},
		{
			name:  "featureGates.tlsProfile given, not using default value",
			input: "../testdata/tlsprofile_old.yaml",
			modifyCheck: func(t *testing.T, cfg configv1alpha1.ProjectConfig) {
				// Config file overrides TLSProfile to "Old"
				assert.Equal(t, configv1alpha1.TLSProfileOldType, cfg.Gates.TLSProfile)
				// Other community defaults should still be present
				assert.Equal(t, "community", cfg.Distribution)
				assert.True(t, cfg.Gates.HTTPEncryption)
			},
		},
		{
			name:  "invalid featureGates.tlsProfile given, show error",
			input: "../testdata/tlsprofile_invalid.yaml",
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
				test.modifyCheck(t, rootCmdConfig.CtrlConfig)
			} else {
				require.Error(t, err)
				require.Equal(t, test.err, err.Error())
			}
		})
	}
}

package cloudcredentials

import (
	"os"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// DiscoverTokenCCOAuthConfig return a token config based on the env variables.
func DiscoverTokenCCOAuthConfig() *manifestutils.TokenCCOAuthConfig {
	// AWS
	roleARN := os.Getenv("ROLEARN")

	switch {
	case roleARN != "":
		return &manifestutils.TokenCCOAuthConfig{
			AWS: &manifestutils.TokenCCOAWSEnvironment{
				RoleARN: roleARN,
			},
		}
	}

	return nil
}

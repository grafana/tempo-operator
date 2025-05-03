package cloudcredentials

import (
	"os"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// DiscoverTokenCCOAuthConfig return a token config based on the env variables.
func DiscoverTokenCCOAuthConfig() (manifestutils.TokenCCOAuthConfig, bool) {
	// AWS
	roleARN := os.Getenv("ROLEARN")

	switch {
	case roleARN != "":
		return manifestutils.TokenCCOAuthConfig{
			AWS: &manifestutils.TokenCCOAWSEnvironment{
				RoleARN: roleARN,
			},
		}, true
	}

	return manifestutils.TokenCCOAuthConfig{}, false
}

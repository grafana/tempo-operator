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
		// The partition of the resources the role may access must match the partition of
		// the role itself, which is the second field of its ARN.
		partition, _ := manifestutils.AWSPartitionForARN(roleARN)
		return &manifestutils.TokenCCOAuthConfig{
			AWS: &manifestutils.TokenCCOAWSEnvironment{
				RoleARN:   roleARN,
				Partition: partition.ID,
			},
		}
	}

	return nil
}

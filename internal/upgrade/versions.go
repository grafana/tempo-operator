package upgrade

import (
	"context"

	"github.com/Masterminds/semver/v3"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

type upgradeFunc func(ctx context.Context, u Upgrade, tempo *v1alpha1.TempoStack) (*v1alpha1.TempoStack, error)

type versionUpgrade struct {
	version semver.Version
	upgrade upgradeFunc
}

var (
	// List of all operator versions requiring "manual" upgrade steps
	// This list needs to be sorted by the version ascending.
	upgrades = []versionUpgrade{
		{
			version: *semver.MustParse("0.1.0"),
			upgrade: upgrade0_1_0,
		},
		{
			version: *semver.MustParse("0.3.0"),
			upgrade: upgrade0_3_0,
		},
	}
)

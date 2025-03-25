package upgrade

import (
	"context"

	"github.com/Masterminds/semver/v3"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

type upgradeTempoStackFn func(ctx context.Context, u Upgrade, tempo *v1alpha1.TempoStack) error
type upgradeTempoMonolithicFn func(ctx context.Context, u Upgrade, tempo *v1alpha1.TempoMonolithic) error

type versionUpgrade struct {
	version semver.Version

	// Optional upgrade function for TempoStack
	upgradeTempoStack upgradeTempoStackFn

	// Optional upgrade function for TempoMonolithic
	upgradeTempoMonolithic upgradeTempoMonolithicFn
}

var (
	// List of all operator versions requiring "manual" upgrade steps
	// This list must be sorted by version ascending.
	upgrades = []versionUpgrade{
		{
			version:           *semver.MustParse("0.1.0"),
			upgradeTempoStack: upgrade0_1_0,
		},
		{
			version:           *semver.MustParse("0.3.0"),
			upgradeTempoStack: upgrade0_3_0,
		},
		{
			version:           *semver.MustParse("0.5.0"),
			upgradeTempoStack: upgrade0_5_0,
		},
		{
			version:           *semver.MustParse("0.6.0"),
			upgradeTempoStack: upgrade0_6_0,
		},
		{
			version:           *semver.MustParse("0.8.0"),
			upgradeTempoStack: upgrade0_8_0,
		},
		{
			version:                *semver.MustParse("0.11.0"),
			upgradeTempoStack:      upgrade0_11_0,
			upgradeTempoMonolithic: upgrade0_11_0_monolithic,
		},
		{
			version:           *semver.MustParse("0.15.4"),
			upgradeTempoStack: upgrade0_15_4,
		},
	}
)

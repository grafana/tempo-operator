package upgrade

import (
	"context"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

// This is a template for future versions.
func upgrade0_1_0(ctx context.Context, u Upgrade, tempo *v1alpha1.TempoStack) error {
	// no-op because 0.1.0 is the first released tempo-operator version
	return nil
}

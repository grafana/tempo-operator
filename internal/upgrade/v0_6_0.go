package upgrade

import (
	"context"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

// This upgrade unsets the image fields in the TempoStack CR.
// From 0.6.0 onwards, the image location is not stored in the CR unless it got changed manually.
func upgrade0_6_0(ctx context.Context, u Upgrade, tempo *v1alpha1.TempoStack) error {
	tempo.Spec.Images.Tempo = ""
	tempo.Spec.Images.TempoQuery = ""
	tempo.Spec.Images.TempoGateway = ""
	tempo.Spec.Images.TempoGatewayOpa = ""
	return nil
}

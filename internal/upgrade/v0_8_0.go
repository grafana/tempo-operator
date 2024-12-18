package upgrade

import (
	"context"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

// The .spec.storage.tls field from TempoStack CR changed from {caName: ""} to TLSSpec
// Set the enabled field if the caName was set previously.
func upgrade0_8_0(ctx context.Context, u Upgrade, tempo *v1alpha1.TempoStack) error {
	if tempo.Spec.Storage.TLS.CA != "" {
		tempo.Spec.Storage.TLS.Enabled = true
	}
	return nil
}

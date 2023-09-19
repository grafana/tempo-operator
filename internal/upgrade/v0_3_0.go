package upgrade

import (
	"context"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

func upgrade0_3_0(ctx context.Context, u Upgrade, tempo *v1alpha1.TempoStack) (*v1alpha1.TempoStack, error) {
	if tempo.Spec.LimitSpec.Global.Query.MaxSearchBytesPerTrace != nil {
		tempo.Spec.LimitSpec.Global.Query.MaxSearchBytesPerTrace = nil
	}
	for tenant, limits := range tempo.Spec.LimitSpec.PerTenant {
		if limits.Query.MaxSearchBytesPerTrace != nil {
			limits.Query.MaxSearchBytesPerTrace = nil
			tempo.Spec.LimitSpec.PerTenant[tenant] = limits
		}
	}
	return tempo, nil
}

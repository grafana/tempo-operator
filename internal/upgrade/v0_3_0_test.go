package upgrade

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/record"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/version"
)

func TestRemoveDeprecatedFields(t *testing.T) {
	zero := 0
	tempo := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			LimitSpec: v1alpha1.LimitSpec{
				Global: v1alpha1.RateLimitSpec{
					Query: v1alpha1.QueryLimit{
						MaxSearchBytesPerTrace: &zero,
					},
				},
				PerTenant: map[string]v1alpha1.RateLimitSpec{
					"tenant1": {
						Query: v1alpha1.QueryLimit{
							MaxSearchBytesPerTrace: &zero,
						},
					},
				},
			},
		},
		Status: v1alpha1.TempoStackStatus{
			OperatorVersion: "0.1.0",
		},
	}

	version := version.Get()
	version.OperatorVersion = "0.3.0"
	upgrade := &Upgrade{
		Client:   k8sClient,
		Recorder: record.NewFakeRecorder(1),
		Version:  version,
		Log:      logger,
	}

	u, err := upgrade.upgradeSpec(context.Background(), &tempo)
	require.NoError(t, err)
	upgraded := u.(*v1alpha1.TempoStack)
	require.Nil(t, upgraded.Spec.LimitSpec.Global.Query.MaxSearchBytesPerTrace)
	require.Nil(t, upgraded.Spec.LimitSpec.PerTenant["tenant1"].Query.MaxSearchBytesPerTrace)
}

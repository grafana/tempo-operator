package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// ingesterLifecycler renders the Tempo configuration and returns the ingester.lifecycler section.
func ingesterLifecycler(t *testing.T, spec v1alpha1.TempoStackSpec) map[string]any {
	t.Helper()

	cfg, err := buildConfiguration(manifestutils.Params{
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "project1"},
			Spec:       spec,
		},
	})
	require.NoError(t, err)

	parsed := map[string]any{}
	require.NoError(t, yaml.Unmarshal(cfg, &parsed))

	ingester, ok := parsed["ingester"].(map[string]any)
	require.True(t, ok, "ingester section not found in %s", string(cfg))
	lifecycler, ok := ingester["lifecycler"].(map[string]any)
	require.True(t, ok, "ingester.lifecycler section not found in %s", string(cfg))
	return lifecycler
}

func TestBuildConfigurationZoneAwarenessDisabled(t *testing.T) {
	for _, tc := range []struct {
		name string
		spec v1alpha1.TempoStackSpec
	}{
		{"no replication spec", v1alpha1.TempoStackSpec{ReplicationFactor: 3}},
		{"no zones", v1alpha1.TempoStackSpec{Replication: &v1alpha1.ReplicationSpec{Factor: 3}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lifecycler := ingesterLifecycler(t, tc.spec)

			assert.NotContains(t, lifecycler, "availability_zone")
			assert.NotContains(t, lifecycler["ring"], "zone_awareness_enabled")
			assert.Equal(t, 3, lifecycler["ring"].(map[string]any)["replication_factor"])
		})
	}
}

func TestBuildConfigurationZoneAwarenessEnabled(t *testing.T) {
	lifecycler := ingesterLifecycler(t, v1alpha1.TempoStackSpec{
		Replication: &v1alpha1.ReplicationSpec{
			Factor: 3,
			Zones:  []v1alpha1.ZoneSpec{{MaxSkew: 1, TopologyKey: "topology.kubernetes.io/zone"}},
		},
	})

	// the value is expanded at startup from the env var set by the operator,
	// the containers run with -config.expand-env=true
	assert.Equal(t, "${INSTANCE_AVAILABILITY_ZONE}", lifecycler["availability_zone"])

	ring := lifecycler["ring"].(map[string]any)
	assert.Equal(t, true, ring["zone_awareness_enabled"])
	assert.Equal(t, 3, ring["replication_factor"])
}

func TestBuildConfigurationReplicationFactorPrecedence(t *testing.T) {
	// spec.replication.factor takes precedence over the deprecated spec.replicationFactor
	lifecycler := ingesterLifecycler(t, v1alpha1.TempoStackSpec{
		ReplicationFactor: 2,
		Replication:       &v1alpha1.ReplicationSpec{Factor: 3},
	})

	assert.Equal(t, 3, lifecycler["ring"].(map[string]any)["replication_factor"])
}

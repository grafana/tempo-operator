package alerts

import (
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPrometheusRule(t *testing.T) {
	objects, err := BuildPrometheusRule("tempo-test", "default")

	require.NoError(t, err)
	assert.Len(t, objects, 1)
	rules := objects[0].(*monitoringv1.PrometheusRule)

	assert.Equal(t, "tempo-test-prometheus-rule", rules.Name)
}

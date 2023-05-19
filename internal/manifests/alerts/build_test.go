package alerts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRules(t *testing.T) {
	rulesSpec, err := build(Options{
		RunbookURL: RunbookDefaultURL,
	})

	require.NoError(t, err)
	assert.Len(t, rulesSpec.Groups, 2)
	assert.Equal(t, "tempo_alerts", rulesSpec.Groups[0].Name)
	assert.Len(t, rulesSpec.Groups[0].Rules, 14)

	assert.Equal(t, "tempo_rules", rulesSpec.Groups[1].Name)
	assert.Len(t, rulesSpec.Groups[1].Rules, 6)

}

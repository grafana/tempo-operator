package networking

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func TestGenerateOperatorPolicies(t *testing.T) {
	//nolint:errcheck
	os.Setenv("ENABLE_WEBHOOKS", "true")
	//nolint:errcheck
	defer os.Unsetenv("ENABLE_WEBHOOKS")

	namespace := "($TEMPO_NAMESPACE)"

	policies := GenerateOperatorPolicies(namespace)

	expectedPolicies, err := loadExpectedPolicies(t, "../../../tests/e2e/networking/00-asserts.yaml")
	require.NoError(t, err, "Failed to load expected policies from YAML")

	assert.Equal(t, expectedPolicies, policies, "Generated policies do not match expected policies")
}

func loadExpectedPolicies(t *testing.T, filePath string) ([]client.Object, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var objs []client.Object
	for _, np := range bytes.Split(data, []byte("---")) {
		policy := &networkingv1.NetworkPolicy{}
		require.NoError(t, yaml.Unmarshal(np, &policy))
		objs = append(objs, policy)
	}
	return objs, nil
}

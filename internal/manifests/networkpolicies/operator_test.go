package networkpolicies

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestGenerateOperatorPolicies(t *testing.T) {
	//nolint:errcheck
	os.Setenv("ENABLE_WEBHOOKS", "true")
	//nolint:errcheck
	defer os.Unsetenv("ENABLE_WEBHOOKS")

	namespace := "($TEMPO_NAMESPACE)"

	// Use fallback API server info for tests (port 6443, no specific IPs, no service IPs)
	apiServerInfo := fallbackKubeAPIServerInfo(nil, nil)

	policies := GenerateOperatorPolicies(namespace, apiServerInfo)

	expectedPolicies, err := loadExpectedPolicies(t, "../../../tests/e2e/networkpolicies-operator/00-asserts.yaml")
	require.NoError(t, err, "Failed to load expected policies from YAML")

	assert.Equal(t, expectedPolicies, policies, "Generated policies do not match expected policies")
}

func TestGenerateOperatorPoliciesWithCustomAPIServerPort(t *testing.T) {
	tests := []struct {
		name              string
		apiServerPort     int32
		apiServerIPs      []string
		expectedPortCount int
		expectedPeerCount int
		expectSpecificIPs bool
	}{
		{
			name:              "EKS with port 443 and specific IPs",
			apiServerPort:     443,
			apiServerIPs:      []string{"100.105.216.2", "100.109.70.105"},
			expectedPortCount: 1,
			expectedPeerCount: 2,
			expectSpecificIPs: true,
		},
		{
			name:              "Standard K8s with port 6443",
			apiServerPort:     6443,
			apiServerIPs:      []string{"10.0.0.1"},
			expectedPortCount: 1,
			expectedPeerCount: 1,
			expectSpecificIPs: true,
		},
		{
			name:              "Fallback without specific IPs",
			apiServerPort:     6443,
			apiServerIPs:      nil,
			expectedPortCount: 1,
			expectedPeerCount: 0,
			expectSpecificIPs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace := "tempo-operator"

			// Create API server info with custom port and IPs
			apiServerInfo := manifestutils.KubeAPIServerInfo{
				Ports: []networkingv1.NetworkPolicyPort{
					{
						Protocol: ptr.To(corev1.ProtocolTCP),
						Port:     ptr.To(intstr.FromInt(int(tt.apiServerPort))),
					},
				},
				IPs: tt.apiServerIPs,
			}

			policies := GenerateOperatorPolicies(namespace, apiServerInfo)

			// Find the API server egress policy
			var apiServerPolicy *networkingv1.NetworkPolicy
			for _, obj := range policies {
				if np, ok := obj.(*networkingv1.NetworkPolicy); ok {
					if strings.Contains(np.Name, "egress-to-apiserver") {
						apiServerPolicy = np
						break
					}
				}
			}

			require.NotNil(t, apiServerPolicy, "API server egress policy not found")

			// Verify the policy has egress rules
			require.Len(t, apiServerPolicy.Spec.Egress, 1)
			egressRule := apiServerPolicy.Spec.Egress[0]

			// Verify port
			require.Len(t, egressRule.Ports, tt.expectedPortCount)
			assert.Equal(t, tt.apiServerPort, egressRule.Ports[0].Port.IntVal,
				"Expected API server port %d, got %d", tt.apiServerPort, egressRule.Ports[0].Port.IntVal)

			// Verify IPs/peers
			if tt.expectSpecificIPs {
				require.Len(t, egressRule.To, tt.expectedPeerCount,
					"Expected %d specific IP peers, got %d", tt.expectedPeerCount, len(egressRule.To))

				// Verify each IP is present as /32 CIDR
				for _, expectedIP := range tt.apiServerIPs {
					found := false
					expectedCIDR := expectedIP + "/32"
					for _, peer := range egressRule.To {
						if peer.IPBlock != nil && peer.IPBlock.CIDR == expectedCIDR {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected IP %s not found in operator NetworkPolicy", expectedIP)
				}
			} else {
				// When no specific IPs, the To field should be empty (allows all destinations)
				assert.Len(t, egressRule.To, 0,
					"Expected no specific peers (allows all destinations), got %d peers", len(egressRule.To))
			}
		})
	}
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

// TestGenerateOperatorPoliciesWithAPIServerServiceIP covers https://github.com/grafana/tempo-operator/issues/1386:
// the operator dials the cluster IP of the kubernetes service, which CNIs evaluating egress
// policies before DNAT see as the destination of the traffic.
func TestGenerateOperatorPoliciesWithAPIServerServiceIP(t *testing.T) {
	apiServerInfo := manifestutils.KubeAPIServerInfo{
		Ports: []networkingv1.NetworkPolicyPort{
			{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt(6443))},
		},
		IPs: []string{"10.0.0.1", "10.0.0.2"},
		ServicePorts: []networkingv1.NetworkPolicyPort{
			{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt(443))},
		},
		ServiceIPs: []string{"10.96.0.1"},
	}

	policy := policyAPIServer("operator", "tempo-operator-system", apiServerInfo)

	require.Len(t, policy.Spec.Egress, 2, "expected one rule for the endpoints and one for the service")

	endpoints := policy.Spec.Egress[0]
	assert.Equal(t, int32(6443), endpoints.Ports[0].Port.IntVal)
	assert.Equal(t, []networkingv1.NetworkPolicyPeer{
		{IPBlock: &networkingv1.IPBlock{CIDR: "10.0.0.1/32"}},
		{IPBlock: &networkingv1.IPBlock{CIDR: "10.0.0.2/32"}},
	}, endpoints.To)

	service := policy.Spec.Egress[1]
	assert.Equal(t, int32(443), service.Ports[0].Port.IntVal)
	assert.Equal(t, []networkingv1.NetworkPolicyPeer{
		{IPBlock: &networkingv1.IPBlock{CIDR: "10.96.0.1/32"}},
	}, service.To)
}

func TestGenerateOperatorPoliciesWithoutAPIServerServiceIP(t *testing.T) {
	apiServerInfo := manifestutils.KubeAPIServerInfo{
		Ports: []networkingv1.NetworkPolicyPort{
			{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt(6443))},
		},
		IPs: []string{"10.0.0.1"},
	}

	policy := policyAPIServer("operator", "tempo-operator-system", apiServerInfo)
	assert.Len(t, policy.Spec.Egress, 1)
}

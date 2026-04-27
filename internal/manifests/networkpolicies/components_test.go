package networkpolicies

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

func TestNetworkPolicy(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myinstance",
			Namespace: "something",
		},
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				Gateway: v1alpha1.TempoGatewaySpec{
					Enabled: true,
				},
				QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
					JaegerQuery: v1alpha1.JaegerQuerySpec{
						Enabled: true,
					},
				},
			},
		},
	}

	params := manifestutils.Params{
		Tempo: tempo,
		StorageParams: manifestutils.StorageParams{
			S3: &manifestutils.S3{
				Endpoint: "minio:9000",
				Bucket:   "tempo",
				Insecure: false,
			},
			CredentialMode: v1alpha1.CredentialModeStatic,
		},
	}

	componentName := manifestutils.IngesterComponentName
	// componentName = manifestutils.DistributorComponentName
	np := generatePolicyFor(params, componentName)

	require.NotNil(t, np)

	assert.Equal(t, np.ObjectMeta.Name, naming.Name(componentName, tempo.Name))
	assert.Equal(t, np.ObjectMeta.Namespace, tempo.Namespace)
	assert.True(t, labels.Equals(manifestutils.ComponentLabels(componentName, tempo.Name), np.Spec.PodSelector.MatchLabels))
	assert.Len(t, np.Spec.PolicyTypes, 2) // Ingester has both Egress (S3) and Ingress (from distributor/querier)
	assert.Len(t, np.Spec.Egress, 1)
	// Storage egress rules don't specify ports because kube-proxy DNATs ClusterIP service
	// traffic to the targetPort, which may differ from the service port. Network policies
	// evaluate after DNAT, so we can't rely on the service port.
	assert.Len(t, np.Spec.Egress[0].Ports, 0)
}

func TestReverseRelations(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]map[string][]networkingv1.NetworkPolicyPort
		expected map[string]map[string][]networkingv1.NetworkPolicyPort
	}{
		{
			name: "Simple case",
			input: map[string]map[string][]networkingv1.NetworkPolicyPort{
				"A": {
					"B": {{Port: ptr.To(intstr.FromInt(80))}},
				},
			},
			expected: map[string]map[string][]networkingv1.NetworkPolicyPort{
				"B": {
					"A": {{Port: ptr.To(intstr.FromInt(80))}},
				},
			},
		},
		{
			name: "Multiple targets",
			input: map[string]map[string][]networkingv1.NetworkPolicyPort{
				"A": {
					"B": {{Port: ptr.To(intstr.FromInt(80))}},
					"C": {{Port: ptr.To(intstr.FromInt(443))}},
				},
			},
			expected: map[string]map[string][]networkingv1.NetworkPolicyPort{
				"B": {
					"A": {{Port: ptr.To(intstr.FromInt(80))}},
				},
				"C": {
					"A": {{Port: ptr.To(intstr.FromInt(443))}},
				},
			},
		},
		{
			name: "Reverse relations",
			input: map[string]map[string][]networkingv1.NetworkPolicyPort{
				"A": {
					"B": {{Port: ptr.To(intstr.FromInt(80))}},
				},
				"B": {
					"A": {{Port: ptr.To(intstr.FromInt(443))}},
				},
			},
			expected: map[string]map[string][]networkingv1.NetworkPolicyPort{
				"A": {
					"B": {{Port: ptr.To(intstr.FromInt(443))}},
				},
				"B": {
					"A": {{Port: ptr.To(intstr.FromInt(80))}},
				},
			},
		},
		{
			name:     "Empty input",
			input:    map[string]map[string][]networkingv1.NetworkPolicyPort{},
			expected: map[string]map[string][]networkingv1.NetworkPolicyPort{},
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reverseRelations(tt.input)
			if !equal(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func equal(a, b map[string]map[string][]networkingv1.NetworkPolicyPort) bool {
	if len(a) != len(b) {
		return false
	}
	for key, valueA := range a {
		valueB, ok := b[key]
		if !ok || len(valueA) != len(valueB) {
			return false
		}
		for subKey, portsA := range valueA {
			portsB, ok := valueB[subKey]
			if !ok || len(portsA) != len(portsB) {
				return false
			}
			for i := range portsA {
				if !compareNetworkPolicyPorts(portsA[i], portsB[i]) {
					return false
				}
			}
		}
	}
	return true
}

func compareNetworkPolicyPorts(a, b networkingv1.NetworkPolicyPort) bool {
	return a.Port.IntValue() == b.Port.IntValue()
}

func TestOAuthServerEgress(t *testing.T) {
	// Test that OAuth server egress (port 443) is added to gateway when gateway is enabled
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myinstance",
			Namespace: "something",
		},
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				Gateway: v1alpha1.TempoGatewaySpec{
					Enabled: true,
				},
			},
		},
	}

	params := manifestutils.Params{
		Tempo: tempo,
		StorageParams: manifestutils.StorageParams{
			S3: &manifestutils.S3{
				Endpoint: "minio:9000",
				Bucket:   "tempo",
				Insecure: false,
			},
			CredentialMode: v1alpha1.CredentialModeStatic,
		},
	}

	np := generatePolicyFor(params, manifestutils.GatewayComponentName)
	require.NotNil(t, np)

	// Verify gateway has OAuth server egress on port 443
	var hasOAuthServerEgress bool
	for _, egress := range np.Spec.Egress {
		for _, port := range egress.Ports {
			if port.Port != nil && port.Port.IntValue() == 443 {
				hasOAuthServerEgress = true
				break
			}
		}
	}
	assert.True(t, hasOAuthServerEgress, "gateway should have OAuth server egress on port 443")

	// Verify egress policy type is set
	assert.Contains(t, np.Spec.PolicyTypes, networkingv1.PolicyTypeEgress)
}

func TestOAuthProxyPort(t *testing.T) {
	tests := []struct {
		name                 string
		authEnabled          bool
		expectOAuthProxyPort bool
	}{
		{
			name:                 "JaegerQuery auth enabled - should include OAuth proxy port",
			authEnabled:          true,
			expectOAuthProxyPort: true,
		},
		{
			name:                 "JaegerQuery auth disabled - should not include OAuth proxy port",
			authEnabled:          false,
			expectOAuthProxyPort: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempo := v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myinstance",
					Namespace: "something",
				},
				Spec: v1alpha1.TempoStackSpec{
					Template: v1alpha1.TempoTemplateSpec{
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: false, // Gateway disabled to test query-frontend directly
						},
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: true,
								Authentication: &v1alpha1.JaegerQueryAuthenticationSpec{
									Enabled: tt.authEnabled,
								},
							},
						},
					},
				},
			}

			params := manifestutils.Params{
				Tempo: tempo,
				StorageParams: manifestutils.StorageParams{
					S3: &manifestutils.S3{
						Endpoint: "minio:9000",
						Bucket:   "tempo",
						Insecure: false,
					},
					CredentialMode: v1alpha1.CredentialModeStatic,
				},
			}

			np := generatePolicyFor(params, manifestutils.QueryFrontendComponentName)
			require.NotNil(t, np)

			// Check for OAuth proxy port (8443) in ingress rules
			var hasOAuthProxyPort bool
			for _, ingress := range np.Spec.Ingress {
				for _, port := range ingress.Ports {
					if port.Port != nil && port.Port.IntValue() == manifestutils.OAuthProxyPort {
						hasOAuthProxyPort = true
						break
					}
				}
			}

			if tt.expectOAuthProxyPort {
				assert.True(t, hasOAuthProxyPort, "query-frontend should have OAuth proxy port (8443) when auth is enabled")
			} else {
				assert.False(t, hasOAuthProxyPort, "query-frontend should not have OAuth proxy port (8443) when auth is disabled")
			}
		})
	}
}

func TestJaegerQueryPorts(t *testing.T) {
	tests := []struct {
		name                 string
		gatewayEnabled       bool
		jaegerEnabled        bool
		expectedPorts        []int
		expectedGatewayPorts []int
	}{
		{
			name:                 "JaegerQuery enabled with gateway - should include all Jaeger ports",
			gatewayEnabled:       true,
			jaegerEnabled:        true,
			expectedPorts:        []int{manifestutils.PortJaegerGRPCQuery, manifestutils.PortJaegerUI, manifestutils.PortJaegerMetrics},
			expectedGatewayPorts: []int{manifestutils.PortJaegerUI, manifestutils.PortJaegerMetrics},
		},
		{
			name:                 "JaegerQuery enabled without gateway - should include all Jaeger ports",
			gatewayEnabled:       false,
			jaegerEnabled:        true,
			expectedPorts:        []int{manifestutils.PortJaegerGRPCQuery, manifestutils.PortJaegerUI, manifestutils.PortJaegerMetrics},
			expectedGatewayPorts: []int{},
		},
		{
			name:                 "JaegerQuery disabled - should not include Jaeger ports",
			gatewayEnabled:       false,
			jaegerEnabled:        false,
			expectedPorts:        []int{},
			expectedGatewayPorts: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempo := v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myinstance",
					Namespace: "something",
				},
				Spec: v1alpha1.TempoStackSpec{
					Template: v1alpha1.TempoTemplateSpec{
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: tt.gatewayEnabled,
						},
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: tt.jaegerEnabled,
							},
						},
					},
				},
			}

			params := manifestutils.Params{
				Tempo: tempo,
				StorageParams: manifestutils.StorageParams{
					S3: &manifestutils.S3{
						Endpoint: "minio:9000",
						Bucket:   "tempo",
						Insecure: false,
					},
					CredentialMode: v1alpha1.CredentialModeStatic,
				},
			}

			// Test query-frontend NetworkPolicy
			np := generatePolicyFor(params, manifestutils.QueryFrontendComponentName)
			require.NotNil(t, np)

			// Check ingress rules for expected Jaeger ports
			foundPorts := make(map[int]bool)
			for _, ingress := range np.Spec.Ingress {
				for _, port := range ingress.Ports {
					if port.Port != nil {
						foundPorts[port.Port.IntValue()] = true
					}
				}
			}

			for _, expectedPort := range tt.expectedPorts {
				assert.True(t, foundPorts[expectedPort],
					"query-frontend should have ingress port %d (JaegerQuery enabled: %v)",
					expectedPort, tt.jaegerEnabled)
			}

			// If gateway is enabled, also check gateway NetworkPolicy
			if tt.gatewayEnabled {
				gwNp := generatePolicyFor(params, manifestutils.GatewayComponentName)
				require.NotNil(t, gwNp)

				// Check egress rules for expected Jaeger ports to query-frontend
				foundGwPorts := make(map[int]bool)
				for _, egress := range gwNp.Spec.Egress {
					for _, port := range egress.Ports {
						if port.Port != nil {
							foundGwPorts[port.Port.IntValue()] = true
						}
					}
				}

				for _, expectedPort := range tt.expectedGatewayPorts {
					assert.True(t, foundGwPorts[expectedPort],
						"gateway should have egress port %d to query-frontend (JaegerQuery enabled: %v)",
						expectedPort, tt.jaegerEnabled)
				}
			}
		})
	}
}

func TestGatewayPorts(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myinstance",
			Namespace: "something",
		},
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				Gateway: v1alpha1.TempoGatewaySpec{
					Enabled: true,
				},
			},
		},
	}

	params := manifestutils.Params{
		Tempo: tempo,
		StorageParams: manifestutils.StorageParams{
			S3: &manifestutils.S3{
				Endpoint: "minio:9000",
				Bucket:   "tempo",
				Insecure: false,
			},
			CredentialMode: v1alpha1.CredentialModeStatic,
		},
	}

	np := generatePolicyFor(params, manifestutils.GatewayComponentName)
	require.NotNil(t, np)

	// Check that all three gateway ports are allowed for ingress from cluster components
	expectedPorts := []int{
		manifestutils.GatewayPortHTTPServer,         // 8080
		manifestutils.GatewayPortInternalHTTPServer, // 8081
		manifestutils.GatewayPortGRPCServer,         // 8090
	}

	foundPorts := make(map[int]bool)
	for _, ingress := range np.Spec.Ingress {
		for _, port := range ingress.Ports {
			if port.Port != nil {
				foundPorts[port.Port.IntValue()] = true
			}
		}
	}

	for _, expectedPort := range expectedPorts {
		assert.True(t, foundPorts[expectedPort],
			"gateway should have ingress port %d", expectedPort)
	}

	// Verify ingress policy type is set
	assert.Contains(t, np.Spec.PolicyTypes, networkingv1.PolicyTypeIngress)
}

func TestMetricsGeneratorPolicy(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myinstance",
			Namespace: "something",
		},
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				MetricsGenerator: &v1alpha1.TempoMetricsGeneratorSpec{
					RemoteWriteURLs: []string{"http://prometheus:9090/api/v1/write"},
				},
			},
		},
	}

	params := manifestutils.Params{
		Tempo: tempo,
		StorageParams: manifestutils.StorageParams{
			S3: &manifestutils.S3{
				Endpoint: "minio:9000",
				Bucket:   "tempo",
				Insecure: false,
			},
			CredentialMode: v1alpha1.CredentialModeStatic,
		},
	}

	np := generatePolicyFor(params, manifestutils.MetricsGeneratorComponentName)
	require.NotNil(t, np)

	assert.Equal(t, naming.Name(manifestutils.MetricsGeneratorComponentName, tempo.Name), np.ObjectMeta.Name)
	assert.Equal(t, tempo.Namespace, np.ObjectMeta.Namespace)
	assert.True(t, labels.Equals(manifestutils.ComponentLabels(manifestutils.MetricsGeneratorComponentName, tempo.Name), np.Spec.PodSelector.MatchLabels))

	// Should have egress (remote write) and ingress (from distributor)
	assert.Contains(t, np.Spec.PolicyTypes, networkingv1.PolicyTypeEgress)
	assert.Contains(t, np.Spec.PolicyTypes, networkingv1.PolicyTypeIngress)

	// Egress to Prometheus on port 9090
	var hasPrometheusEgress bool
	for _, egress := range np.Spec.Egress {
		for _, port := range egress.Ports {
			if port.Port != nil && port.Port.IntValue() == manifestutils.PortPrometheusServer {
				hasPrometheusEgress = true
				break
			}
		}
	}
	assert.True(t, hasPrometheusEgress, "metrics-generator should have egress to Prometheus on port 9090")

	// Ingress from distributor on gRPC/HTTP ports
	var hasDistributorIngress bool
	for _, ingress := range np.Spec.Ingress {
		for _, port := range ingress.Ports {
			if port.Port != nil && port.Port.IntValue() == manifestutils.PortGRPCServer {
				hasDistributorIngress = true
				break
			}
		}
	}
	assert.True(t, hasDistributorIngress, "metrics-generator should accept ingress from distributor on gRPC port")
}

func TestExtractStoragePorts(t *testing.T) {
	tests := []struct {
		name           string
		storageParams  manifestutils.StorageParams
		expectedPort   int
		expectedLength int
	}{
		// S3 tests
		{
			name: "S3 with port 9000",
			storageParams: manifestutils.StorageParams{
				S3: &manifestutils.S3{
					Endpoint: "minio:9000",
					Insecure: true,
				},
				CredentialMode: v1alpha1.CredentialModeStatic,
			},
			expectedPort:   9000,
			expectedLength: 1,
		},
		{
			name: "S3 with port 9443",
			storageParams: manifestutils.StorageParams{
				S3: &manifestutils.S3{
					Endpoint: "s3.example.com:9443",
					Insecure: false,
				},
				CredentialMode: v1alpha1.CredentialModeStatic,
			},
			expectedPort:   9443,
			expectedLength: 1,
		},
		{
			name: "S3 without port (HTTPS)",
			storageParams: manifestutils.StorageParams{
				S3: &manifestutils.S3{
					Endpoint: "s3.example.com",
					Insecure: false,
				},
				CredentialMode: v1alpha1.CredentialModeStatic,
			},
			expectedPort:   443,
			expectedLength: 1,
		},
		{
			name: "S3 without port (HTTP)",
			storageParams: manifestutils.StorageParams{
				S3: &manifestutils.S3{
					Endpoint: "s3.example.com",
					Insecure: true,
				},
				CredentialMode: v1alpha1.CredentialModeStatic,
			},
			expectedPort:   80,
			expectedLength: 1,
		},
		{
			name: "S3 token mode (AWS)",
			storageParams: manifestutils.StorageParams{
				S3: &manifestutils.S3{
					Region: "us-east-1",
				},
				CredentialMode: v1alpha1.CredentialModeToken,
			},
			expectedPort:   443,
			expectedLength: 1,
		},
		// Azure Storage tests
		{
			name: "Azure Storage configured",
			storageParams: manifestutils.StorageParams{
				AzureStorage: &manifestutils.AzureStorage{
					Container: "tempo-traces",
				},
				CredentialMode: v1alpha1.CredentialModeStatic,
			},
			expectedPort:   443,
			expectedLength: 1,
		},
		// GCS tests
		{
			name: "GCS configured",
			storageParams: manifestutils.StorageParams{
				GCS: &manifestutils.GCS{
					Bucket: "tempo-traces",
				},
				CredentialMode: v1alpha1.CredentialModeStatic,
			},
			expectedPort:   443,
			expectedLength: 1,
		},
		// No storage configured
		{
			name: "No storage configured",
			storageParams: manifestutils.StorageParams{
				S3:           nil,
				AzureStorage: nil,
				GCS:          nil,
			},
			expectedPort:   0,
			expectedLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ports := extractStoragePorts(tt.storageParams)
			assert.Equal(t, tt.expectedLength, len(ports))
			if tt.expectedLength > 0 {
				assert.Equal(t, tt.expectedPort, ports[0].Port.IntValue())
			}
		})
	}
}

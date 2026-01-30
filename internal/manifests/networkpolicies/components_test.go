package networkpolicies

import (
	"fmt"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	got, _ := yaml.Marshal(np)
	fmt.Println(string(got))
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

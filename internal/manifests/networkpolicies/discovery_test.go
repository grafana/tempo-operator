package networkpolicies

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDiscoverKubernetesAPIServer(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, discoveryv1.AddToScheme(scheme))

	t.Run("discovers API server from EndpointSlice", func(t *testing.T) {
		endpointSlice := &discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubernetes",
				Namespace: "default",
				Labels: map[string]string{
					"kubernetes.io/service-name": "kubernetes",
				},
			},
			Ports: []discoveryv1.EndpointPort{
				{
					Name: ptr.To("https"),
					Port: ptr.To(int32(6443)),
				},
			},
			Endpoints: []discoveryv1.Endpoint{
				{
					Addresses: []string{"10.0.0.1"},
				},
				{
					Addresses: []string{"10.0.0.2"},
				},
			},
		}

		client := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(endpointSlice).
			Build()

		ctx := context.Background()
		info := DiscoverKubernetesAPIServer(ctx, client)

		require.Len(t, info.Ports, 1)
		assert.Equal(t, int32(6443), info.Ports[0].Port.IntVal)
		require.Len(t, info.IPs, 2)
		assert.Contains(t, info.IPs, "10.0.0.1")
		assert.Contains(t, info.IPs, "10.0.0.2")
	})

	t.Run("handles port 443 instead of 6443", func(t *testing.T) {
		endpointSlice := &discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubernetes",
				Namespace: "default",
				Labels: map[string]string{
					"kubernetes.io/service-name": "kubernetes",
				},
			},
			Ports: []discoveryv1.EndpointPort{
				{
					Name: ptr.To("https"),
					Port: ptr.To(int32(443)),
				},
			},
			Endpoints: []discoveryv1.Endpoint{
				{
					Addresses: []string{"100.105.216.2"},
				},
			},
		}

		client := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(endpointSlice).
			Build()

		ctx := context.Background()
		info := DiscoverKubernetesAPIServer(ctx, client)

		require.Len(t, info.Ports, 1)
		assert.Equal(t, int32(443), info.Ports[0].Port.IntVal)
		require.Len(t, info.IPs, 1)
		assert.Equal(t, "100.105.216.2", info.IPs[0])
	})

	t.Run("falls back to defaults when EndpointSlice not found", func(t *testing.T) {
		client := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		ctx := context.Background()
		info := DiscoverKubernetesAPIServer(ctx, client)

		// Should fall back to port 6443
		require.Len(t, info.Ports, 1)
		assert.Equal(t, int32(6443), info.Ports[0].Port.IntVal)
		// No specific IPs means allow all with port restriction
		assert.Nil(t, info.IPs)
	})

	t.Run("deduplicates ports and IPs", func(t *testing.T) {
		endpointSlice := &discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubernetes",
				Namespace: "default",
				Labels: map[string]string{
					"kubernetes.io/service-name": "kubernetes",
				},
			},
			Ports: []discoveryv1.EndpointPort{
				{
					Name: ptr.To("https"),
					Port: ptr.To(int32(6443)),
				},
				{
					Name: ptr.To("https-alt"),
					Port: ptr.To(int32(6443)), // Duplicate port
				},
			},
			Endpoints: []discoveryv1.Endpoint{
				{
					Addresses: []string{"10.0.0.1"},
				},
				{
					Addresses: []string{"10.0.0.1"}, // Duplicate IP
				},
				{
					Addresses: []string{"10.0.0.2"},
				},
			},
		}

		client := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(endpointSlice).
			Build()

		ctx := context.Background()
		info := DiscoverKubernetesAPIServer(ctx, client)

		// Should deduplicate to 1 port and 2 IPs
		assert.Len(t, info.Ports, 1)
		assert.Len(t, info.IPs, 2)
	})
}

func TestIsAPIServerEndpointSlice(t *testing.T) {
	endpointSlice := func(namespace string, labels map[string]string) *discoveryv1.EndpointSlice {
		return &discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{Name: "slice", Namespace: namespace, Labels: labels},
		}
	}
	apiServerLabels := map[string]string{"kubernetes.io/service-name": "kubernetes"}

	tests := []struct {
		name     string
		obj      *discoveryv1.EndpointSlice
		expected bool
	}{
		{
			name:     "EndpointSlice of the API server",
			obj:      endpointSlice("default", apiServerLabels),
			expected: true,
		},
		{
			name:     "another service in the default namespace",
			obj:      endpointSlice("default", map[string]string{"kubernetes.io/service-name": "other"}),
			expected: false,
		},
		{
			name:     "same service name in another namespace",
			obj:      endpointSlice("kube-system", apiServerLabels),
			expected: false,
		},
		{
			name:     "no labels",
			obj:      endpointSlice("default", nil),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsAPIServerEndpointSlice(tt.obj))
		})
	}
}

func TestEndpointSliceIPsAndPorts(t *testing.T) {
	t.Run("deduplicates and sorts IPs and ports", func(t *testing.T) {
		endpointSlice := &discoveryv1.EndpointSlice{
			Ports: []discoveryv1.EndpointPort{
				{Port: ptr.To(int32(6443))},
				{Port: ptr.To(int32(443))},
				{Port: ptr.To(int32(6443))},
				{Port: nil}, // a port without a number is ignored
			},
			Endpoints: []discoveryv1.Endpoint{
				{Addresses: []string{"10.0.0.2", "10.0.0.1"}},
				{Addresses: []string{"10.0.0.1"}},
			},
		}

		ips, ports := EndpointSliceIPsAndPorts(endpointSlice)
		assert.Equal(t, []string{"10.0.0.1", "10.0.0.2"}, ips)
		assert.Equal(t, []int32{443, 6443}, ports)
	})

	t.Run("returns the same result regardless of the order of the endpoints", func(t *testing.T) {
		ips, ports := EndpointSliceIPsAndPorts(&discoveryv1.EndpointSlice{
			Ports:     []discoveryv1.EndpointPort{{Port: ptr.To(int32(6443))}},
			Endpoints: []discoveryv1.Endpoint{{Addresses: []string{"10.0.0.1"}}, {Addresses: []string{"10.0.0.2"}}},
		})
		reorderedIPs, reorderedPorts := EndpointSliceIPsAndPorts(&discoveryv1.EndpointSlice{
			Ports:     []discoveryv1.EndpointPort{{Port: ptr.To(int32(6443))}},
			Endpoints: []discoveryv1.Endpoint{{Addresses: []string{"10.0.0.2"}}, {Addresses: []string{"10.0.0.1"}}},
		})

		assert.Equal(t, ips, reorderedIPs)
		assert.Equal(t, ports, reorderedPorts)
	})

	t.Run("returns nothing for another object", func(t *testing.T) {
		ips, ports := EndpointSliceIPsAndPorts(&corev1.Pod{})
		assert.Nil(t, ips)
		assert.Nil(t, ports)
	})
}

package networkpolicies

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

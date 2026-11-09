package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func apiServerEndpointSlice(ips []string, ports []int32) *discoveryv1.EndpointSlice {
	endpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes",
			Namespace: "default",
			Labels:    map[string]string{"kubernetes.io/service-name": "kubernetes"},
		},
	}
	for _, ip := range ips {
		endpointSlice.Endpoints = append(endpointSlice.Endpoints, discoveryv1.Endpoint{Addresses: []string{ip}})
	}
	for _, port := range ports {
		endpointSlice.Ports = append(endpointSlice.Ports, discoveryv1.EndpointPort{Port: ptr.To(port)})
	}
	return endpointSlice
}

func TestAPIServerEndpointSliceChanged(t *testing.T) {
	t.Run("create and delete of the API server EndpointSlice", func(t *testing.T) {
		endpointSlice := apiServerEndpointSlice([]string{"10.0.0.1"}, []int32{6443})

		assert.True(t, apiServerEndpointSliceChanged.Create(event.CreateEvent{Object: endpointSlice}))
		assert.True(t, apiServerEndpointSliceChanged.Delete(event.DeleteEvent{Object: endpointSlice}))
	})

	t.Run("ignores EndpointSlices of other services", func(t *testing.T) {
		otherService := func(ips []string) *discoveryv1.EndpointSlice {
			endpointSlice := apiServerEndpointSlice(ips, []int32{6443})
			endpointSlice.Labels["kubernetes.io/service-name"] = "tempo-simplest-gateway"
			return endpointSlice
		}

		assert.False(t, apiServerEndpointSliceChanged.Create(event.CreateEvent{Object: otherService([]string{"10.0.0.1"})}))
		assert.False(t, apiServerEndpointSliceChanged.Delete(event.DeleteEvent{Object: otherService([]string{"10.0.0.1"})}))
		assert.False(t, apiServerEndpointSliceChanged.Update(event.UpdateEvent{
			ObjectOld: otherService([]string{"10.0.0.1"}),
			ObjectNew: otherService([]string{"10.0.0.9"}),
		}))
	})

	tests := []struct {
		name     string
		old      *discoveryv1.EndpointSlice
		new      *discoveryv1.EndpointSlice
		expected bool
	}{
		{
			name:     "an IP address changed",
			old:      apiServerEndpointSlice([]string{"10.0.0.1", "10.0.0.2"}, []int32{6443}),
			new:      apiServerEndpointSlice([]string{"10.0.0.1", "10.0.0.3"}, []int32{6443}),
			expected: true,
		},
		{
			name:     "a control plane node was added",
			old:      apiServerEndpointSlice([]string{"10.0.0.1"}, []int32{6443}),
			new:      apiServerEndpointSlice([]string{"10.0.0.1", "10.0.0.2"}, []int32{6443}),
			expected: true,
		},
		{
			name:     "the port changed",
			old:      apiServerEndpointSlice([]string{"10.0.0.1"}, []int32{6443}),
			new:      apiServerEndpointSlice([]string{"10.0.0.1"}, []int32{443}),
			expected: true,
		},
		{
			name:     "nothing relevant changed",
			old:      apiServerEndpointSlice([]string{"10.0.0.1", "10.0.0.2"}, []int32{6443}),
			new:      apiServerEndpointSlice([]string{"10.0.0.1", "10.0.0.2"}, []int32{6443}),
			expected: false,
		},
		{
			name:     "the endpoints were only reordered",
			old:      apiServerEndpointSlice([]string{"10.0.0.1", "10.0.0.2"}, []int32{6443}),
			new:      apiServerEndpointSlice([]string{"10.0.0.2", "10.0.0.1"}, []int32{6443}),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changed := apiServerEndpointSliceChanged.Update(event.UpdateEvent{ObjectOld: tt.old, ObjectNew: tt.new})
			assert.Equal(t, tt.expected, changed)
		})
	}
}

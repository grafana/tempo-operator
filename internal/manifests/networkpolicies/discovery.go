package networkpolicies

import (
	"context"
	"maps"
	"slices"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

const (
	// apiServerNamespace is the namespace of the service of the Kubernetes API server.
	apiServerNamespace = "default"
	// apiServerServiceName is the name of the service of the Kubernetes API server.
	apiServerServiceName = "kubernetes"
	// serviceNameLabel is set by Kubernetes on every EndpointSlice and holds the name of the owning service.
	serviceNameLabel = "kubernetes.io/service-name"
)

// IsAPIServerEndpointSlice returns true if the given object is an EndpointSlice of the
// Kubernetes API server service.
func IsAPIServerEndpointSlice(obj client.Object) bool {
	return obj.GetNamespace() == apiServerNamespace &&
		obj.GetLabels()[serviceNameLabel] == apiServerServiceName
}

// EndpointSliceIPsAndPorts returns the deduplicated and sorted IP addresses and ports of an
// EndpointSlice. These are the only parts of the slice which end up in the NetworkPolicies,
// therefore two slices with an equal result produce identical policies.
// It returns nothing if the object is not an EndpointSlice.
func EndpointSliceIPsAndPorts(obj client.Object) ([]string, []int32) {
	endpointSlice, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		return nil, nil
	}

	ipSet := map[string]struct{}{}
	for _, endpoint := range endpointSlice.Endpoints {
		for _, addr := range endpoint.Addresses {
			ipSet[addr] = struct{}{}
		}
	}

	portSet := map[int32]struct{}{}
	for _, port := range endpointSlice.Ports {
		if port.Port != nil {
			portSet[*port.Port] = struct{}{}
		}
	}

	// Sort both, so that a reordering of the endpoints by the API server does not
	// produce a different NetworkPolicy.
	return slices.Sorted(maps.Keys(ipSet)), slices.Sorted(maps.Keys(portSet))
}

// DiscoverKubernetesAPIServer discovers the Kubernetes API server endpoints and ports
// from the EndpointSlice in the default namespace.
// It returns both the ports and IP addresses that can be used in NetworkPolicies.
// If discovery fails, it returns a fallback configuration with port 6443 and CIDR 0.0.0.0/0.
func DiscoverKubernetesAPIServer(ctx context.Context, k8sClient client.Client) manifestutils.KubeAPIServerInfo {
	logger := log.FromContext(ctx)

	// List EndpointSlices for the kubernetes service
	endpointSliceList := &discoveryv1.EndpointSliceList{}
	err := k8sClient.List(ctx, endpointSliceList, &client.ListOptions{
		Namespace: apiServerNamespace,
	})
	if err != nil {
		logger.Error(err, "failed to list EndpointSlices, falling back to default API server port 6443")
		return fallbackKubeAPIServerInfo()
	}

	// Find the kubernetes service EndpointSlice
	var kubeEndpointSlice *discoveryv1.EndpointSlice
	for i := range endpointSliceList.Items {
		if IsAPIServerEndpointSlice(&endpointSliceList.Items[i]) {
			kubeEndpointSlice = &endpointSliceList.Items[i]
			break
		}
	}

	if kubeEndpointSlice == nil {
		logger.Info("kubernetes EndpointSlice not found, falling back to default API server port 6443")
		return fallbackKubeAPIServerInfo()
	}

	ips, portNumbers := EndpointSliceIPsAndPorts(kubeEndpointSlice)

	var ports []networkingv1.NetworkPolicyPort
	for _, port := range portNumbers {
		ports = append(ports, networkingv1.NetworkPolicyPort{
			Protocol: ptr.To(corev1.ProtocolTCP),
			Port:     ptr.To(intstr.FromInt(int(port))),
		})
	}

	// If no ports or IPs found, fall back to defaults
	if len(ports) == 0 || len(ips) == 0 {
		logger.Info("no ports or IPs found in kubernetes EndpointSlice, falling back to defaults",
			"portsFound", len(ports), "ipsFound", len(ips))
		return fallbackKubeAPIServerInfo()
	}

	logger.V(1).Info("discovered Kubernetes API server endpoints",
		"ports", len(ports), "ips", len(ips))

	return manifestutils.KubeAPIServerInfo{
		Ports: ports,
		IPs:   ips,
	}
}

// fallbackKubeAPIServerInfo returns the default API server configuration
// used when discovery fails.
func fallbackKubeAPIServerInfo() manifestutils.KubeAPIServerInfo {
	return manifestutils.KubeAPIServerInfo{
		Ports: []networkingv1.NetworkPolicyPort{
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(6443)),
			},
		},
		IPs: nil, // nil IPs means we'll use 0.0.0.0/0 CIDR
	}
}

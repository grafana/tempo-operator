package networkpolicies

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// DiscoverKubernetesAPIServer discovers the Kubernetes API server endpoints and ports
// from the EndpointSlice in the default namespace.
// It returns both the ports and IP addresses that can be used in NetworkPolicies.
// If discovery fails, it returns a fallback configuration with port 6443 and CIDR 0.0.0.0/0.
func DiscoverKubernetesAPIServer(ctx context.Context, k8sClient client.Client) manifestutils.KubeAPIServerInfo {
	logger := log.FromContext(ctx)

	// List EndpointSlices for the kubernetes service
	endpointSliceList := &discoveryv1.EndpointSliceList{}
	err := k8sClient.List(ctx, endpointSliceList, &client.ListOptions{
		Namespace: "default",
	})
	if err != nil {
		logger.Error(err, "failed to list EndpointSlices, falling back to default API server port 6443")
		return fallbackKubeAPIServerInfo()
	}

	// Find the kubernetes service EndpointSlice
	var kubeEndpointSlice *discoveryv1.EndpointSlice
	for i := range endpointSliceList.Items {
		if endpointSliceList.Items[i].Labels["kubernetes.io/service-name"] == "kubernetes" {
			kubeEndpointSlice = &endpointSliceList.Items[i]
			break
		}
	}

	if kubeEndpointSlice == nil {
		logger.Info("kubernetes EndpointSlice not found, falling back to default API server port 6443")
		return fallbackKubeAPIServerInfo()
	}

	// Extract ports from the EndpointSlice
	var ports []networkingv1.NetworkPolicyPort
	portSet := make(map[int32]bool) // Use a set to deduplicate ports

	for _, port := range kubeEndpointSlice.Ports {
		if port.Port != nil && !portSet[*port.Port] {
			portSet[*port.Port] = true
			ports = append(ports, networkingv1.NetworkPolicyPort{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(int(*port.Port))),
			})
		}
	}

	// Extract IP addresses from the endpoints
	var ips []string
	ipSet := make(map[string]bool) // Use a set to deduplicate IPs

	for _, endpoint := range kubeEndpointSlice.Endpoints {
		for _, addr := range endpoint.Addresses {
			if !ipSet[addr] {
				ipSet[addr] = true
				ips = append(ips, addr)
			}
		}
	}

	// If no ports or IPs found, fall back to defaults
	if len(ports) == 0 || len(ips) == 0 {
		logger.Info("no ports or IPs found in kubernetes EndpointSlice, falling back to defaults",
			"portsFound", len(ports), "ipsFound", len(ips))
		return fallbackKubeAPIServerInfo()
	}

	logger.Info("discovered Kubernetes API server endpoints",
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

package networkpolicies

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// DiscoverKubernetesAPIServer discovers the Kubernetes API server endpoints and ports
// from both the kubernetes Service and EndpointSlice in the default namespace.
// It returns both the ports and IP addresses that can be used in NetworkPolicies.
// This includes both the Service ClusterIP(s) and the endpoint IPs to ensure
// NetworkPolicy rules work correctly before kube-proxy NAT translation occurs.
// If discovery fails, it returns a fallback configuration with port 6443 and CIDR 0.0.0.0/0.
func DiscoverKubernetesAPIServer(ctx context.Context, k8sClient client.Client) manifestutils.KubeAPIServerInfo {
	logger := log.FromContext(ctx)

	// Discover the kubernetes Service ClusterIP
	kubeService := &corev1.Service{}
	svcErr := k8sClient.Get(ctx, types.NamespacedName{
		Namespace: "default",
		Name:      "kubernetes",
	}, kubeService)

	// Extract ClusterIP(s) with dual-stack support
	var clusterIPs []string
	if svcErr == nil {
		// Prefer ClusterIPs slice (supports dual-stack IPv4/IPv6)
		if len(kubeService.Spec.ClusterIPs) > 0 {
			for _, ip := range kubeService.Spec.ClusterIPs {
				if ip != "" && ip != "None" {
					clusterIPs = append(clusterIPs, ip)
				}
			}
		} else if kubeService.Spec.ClusterIP != "" && kubeService.Spec.ClusterIP != "None" {
			clusterIPs = append(clusterIPs, kubeService.Spec.ClusterIP)
		}
	}

	// Extract ports from Service as fallback
	var servicePorts []networkingv1.NetworkPolicyPort
	servicePortSet := make(map[int32]bool)
	if svcErr == nil {
		for _, port := range kubeService.Spec.Ports {
			if port.Port > 0 && !servicePortSet[port.Port] {
				servicePortSet[port.Port] = true
				servicePorts = append(servicePorts, networkingv1.NetworkPolicyPort{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     ptr.To(intstr.FromInt(int(port.Port))),
				})
			}
		}
	}

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

	// Extract ports from the EndpointSlice
	var ports []networkingv1.NetworkPolicyPort
	portSet := make(map[int32]bool) // Use a set to deduplicate ports

	if kubeEndpointSlice != nil {
		for _, port := range kubeEndpointSlice.Ports {
			if port.Port != nil && !portSet[*port.Port] {
				portSet[*port.Port] = true
				ports = append(ports, networkingv1.NetworkPolicyPort{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     ptr.To(intstr.FromInt(int(*port.Port))),
				})
			}
		}
	}

	// Use Service ports if EndpointSlice ports not available
	if len(ports) == 0 && len(servicePorts) > 0 {
		ports = servicePorts
	}

	// Extract IP addresses from the endpoints
	var ips []string
	ipSet := make(map[string]bool) // Use a set to deduplicate IPs

	// Add ClusterIPs to the IP set first
	for _, ip := range clusterIPs {
		if !ipSet[ip] {
			ipSet[ip] = true
			ips = append(ips, ip)
		}
	}

	if kubeEndpointSlice != nil {
		for _, endpoint := range kubeEndpointSlice.Endpoints {
			for _, addr := range endpoint.Addresses {
				if !ipSet[addr] {
					ipSet[addr] = true
					ips = append(ips, addr)
				}
			}
		}
	}

	// Accept partial success: allow ClusterIPs even if no endpoints
	if len(ports) == 0 || (len(ips) == 0 && len(clusterIPs) == 0) {
		logger.Info("insufficient discovery data, falling back to defaults",
			"portsFound", len(ports), "ipsFound", len(ips), "clusterIPsFound", len(clusterIPs))
		return fallbackKubeAPIServerInfo()
	}

	logger.Info("discovered Kubernetes API server endpoints",
		"ports", len(ports),
		"totalIPs", len(ips),
		"clusterIPs", len(clusterIPs))

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

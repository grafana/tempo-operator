package networkpolicies

import (
	"context"
	"strings"

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
)

// ipToCIDR returns the single address CIDR of an IP address.
func ipToCIDR(ip string) string {
	if strings.Contains(ip, ":") {
		return ip + "/128"
	}
	return ip + "/32"
}

// discoverKubernetesService returns the cluster IP addresses and the ports of the kubernetes
// service, through which clients reach the API server.
// It returns nothing if the service cannot be read, in which case only the endpoints of the
// API server are allowed in the NetworkPolicies, as before.
func discoverKubernetesService(ctx context.Context, k8sClient client.Client) ([]string, []networkingv1.NetworkPolicyPort) {
	logger := log.FromContext(ctx)

	service := &corev1.Service{}
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: apiServerNamespace, Name: apiServerServiceName}, service)
	if err != nil {
		logger.Error(err, "failed to get the kubernetes service, the cluster IP of the API server will not be allowed in the NetworkPolicies")
		return nil, nil
	}

	// clusterIPs holds one address per IP family, and is empty for a headless service.
	var ips []string
	for _, ip := range service.Spec.ClusterIPs {
		if ip != "" && ip != corev1.ClusterIPNone {
			ips = append(ips, ip)
		}
	}

	var ports []networkingv1.NetworkPolicyPort
	for _, port := range service.Spec.Ports {
		ports = append(ports, networkingv1.NetworkPolicyPort{
			Protocol: ptr.To(corev1.ProtocolTCP),
			Port:     ptr.To(intstr.FromInt(int(port.Port))),
		})
	}

	if len(ips) == 0 || len(ports) == 0 {
		logger.Info("no cluster IP or port found on the kubernetes service",
			"ipsFound", len(ips), "portsFound", len(ports))
		return nil, nil
	}

	return ips, ports
}

// DiscoverKubernetesAPIServer discovers the Kubernetes API server endpoints and ports
// from the EndpointSlice in the default namespace, and the cluster IP and ports of the
// kubernetes service.
// It returns both the ports and IP addresses that can be used in NetworkPolicies.
// If discovery fails, it returns a fallback configuration with port 6443 and CIDR 0.0.0.0/0.
func DiscoverKubernetesAPIServer(ctx context.Context, k8sClient client.Client) manifestutils.KubeAPIServerInfo {
	logger := log.FromContext(ctx)

	serviceIPs, servicePorts := discoverKubernetesService(ctx, k8sClient)

	// List EndpointSlices for the kubernetes service
	endpointSliceList := &discoveryv1.EndpointSliceList{}
	err := k8sClient.List(ctx, endpointSliceList, &client.ListOptions{
		Namespace: apiServerNamespace,
	})
	if err != nil {
		logger.Error(err, "failed to list EndpointSlices, falling back to default API server port 6443")
		return fallbackKubeAPIServerInfo(serviceIPs, servicePorts)
	}

	// Find the kubernetes service EndpointSlice
	var kubeEndpointSlice *discoveryv1.EndpointSlice
	for i := range endpointSliceList.Items {
		if endpointSliceList.Items[i].Labels["kubernetes.io/service-name"] == apiServerServiceName {
			kubeEndpointSlice = &endpointSliceList.Items[i]
			break
		}
	}

	if kubeEndpointSlice == nil {
		logger.Info("kubernetes EndpointSlice not found, falling back to default API server port 6443")
		return fallbackKubeAPIServerInfo(serviceIPs, servicePorts)
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
		return fallbackKubeAPIServerInfo(serviceIPs, servicePorts)
	}

	logger.V(1).Info("discovered Kubernetes API server endpoints",
		"ports", len(ports), "ips", len(ips),
		"servicePorts", len(servicePorts), "serviceIPs", len(serviceIPs))

	return manifestutils.KubeAPIServerInfo{
		Ports:        ports,
		IPs:          ips,
		ServiceIPs:   serviceIPs,
		ServicePorts: servicePorts,
	}
}

// fallbackKubeAPIServerInfo returns the default API server configuration
// used when discovery of the endpoints fails.
func fallbackKubeAPIServerInfo(serviceIPs []string, servicePorts []networkingv1.NetworkPolicyPort) manifestutils.KubeAPIServerInfo {
	return manifestutils.KubeAPIServerInfo{
		Ports: []networkingv1.NetworkPolicyPort{
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(6443)),
			},
		},
		IPs:          nil, // nil IPs means we'll use 0.0.0.0/0 CIDR
		ServiceIPs:   serviceIPs,
		ServicePorts: servicePorts,
	}
}

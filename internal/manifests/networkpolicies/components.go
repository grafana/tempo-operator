package networkpolicies

import (
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

func generatePolicyFor(params manifestutils.Params, componentName string) *networkingv1.NetworkPolicy {
	tempo := params.Tempo
	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(componentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    manifestutils.ComponentLabels(componentName, tempo.Name),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: manifestutils.ComponentLabels(componentName, tempo.Name),
			},
		},
	}

	rels := componentRelations(params)[componentName]
	// Sort target names to ensure deterministic ordering
	var targets []string
	for target := range rels {
		targets = append(targets, target)
	}
	sort.Strings(targets)

	for _, target := range targets {
		ports := rels[target]
		peers := policyPeersFor(target, params)

		// For storage targets (S3, Azure, GCS), don't specify ports.
		// When using ClusterIP services, kube-proxy DNATs traffic to the targetPort
		// which may differ from the service port. Network policies evaluate after DNAT,
		// so they see the targetPort, not the service port. Since we can't know the
		// targetPort of in-cluster storage services, we allow any port to storage destinations.
		if target == netPolicys3Storage {
			np.Spec.Egress = append(np.Spec.Egress, networkingv1.NetworkPolicyEgressRule{
				To: peers,
			})
			continue
		}

		for _, conn := range ports {
			np.Spec.Egress = append(np.Spec.Egress, networkingv1.NetworkPolicyEgressRule{
				Ports: []networkingv1.NetworkPolicyPort{conn},
				To:    peers,
			})
		}
	}

	if len(np.Spec.Egress) > 0 {
		np.Spec.PolicyTypes = append(np.Spec.PolicyTypes, networkingv1.PolicyTypeEgress)
	}

	reverse := reverseRelations(componentRelations(params))
	// Sort source names to ensure deterministic ordering
	var sources []string
	for source := range reverse {
		sources = append(sources, source)
	}
	sort.Strings(sources)

	for _, source := range sources {
		if source != componentName {
			continue
		}
		ports := reverse[source]
		// Sort target names within each source
		var ingressTargets []string
		for target := range ports {
			ingressTargets = append(ingressTargets, target)
		}
		sort.Strings(ingressTargets)

		for _, target := range ingressTargets {
			conn := ports[target]
			peers := policyPeersFor(target, params)
			np.Spec.Ingress = append(np.Spec.Ingress, networkingv1.NetworkPolicyIngressRule{
				Ports: conn,
				From:  peers,
			})
		}
	}

	if len(np.Spec.Ingress) > 0 {
		np.Spec.PolicyTypes = append(np.Spec.PolicyTypes, networkingv1.PolicyTypeIngress)
	}

	return np
}

func policyPeersFor(name string, params manifestutils.Params) []networkingv1.NetworkPolicyPeer {
	tempo := params.Tempo
	switch name {
	case netPolicyOtelTargets:
		return []networkingv1.NetworkPolicyPeer{
			{
				IPBlock: &networkingv1.IPBlock{CIDR: "0.0.0.0/0"},
			},
		}
	case netPolicys3Storage:
		// Allow egress to S3 storage backends:
		// 1. ipBlock: Enables access to external storage (AWS S3, Azure, GCS) and pod IPs
		// 2. namespaceSelector: Enables access to ClusterIP services in other namespaces (e.g., OpenShift ODF, cross-namespace MinIO)
		// Port restriction provides security - only the configured storage port is allowed
		return []networkingv1.NetworkPolicyPeer{
			{
				IPBlock: &networkingv1.IPBlock{CIDR: "0.0.0.0/0"},
			},
			{
				NamespaceSelector: &metav1.LabelSelector{},
			},
		}
	case netPolicyClusterComponents:
		return []networkingv1.NetworkPolicyPeer{
			{
				NamespaceSelector: &metav1.LabelSelector{},
			},
		}
	case netPolicyKubeAPIServer:
		// Allow egress to Kubernetes API server.
		// Use discovered IPs from EndpointSlice if available, otherwise fall back to 0.0.0.0/0.
		// This works across all Kubernetes distributions (EKS, GKE, standard K8s, OpenShift)
		// where API server location and port vary.
		if len(params.KubeAPIServer.IPs) > 0 {
			// Use specific IP addresses discovered from EndpointSlice
			peers := make([]networkingv1.NetworkPolicyPeer, 0, len(params.KubeAPIServer.IPs))
			for _, ip := range params.KubeAPIServer.IPs {
				peers = append(peers, networkingv1.NetworkPolicyPeer{
					IPBlock: &networkingv1.IPBlock{CIDR: ip + "/32"},
				})
			}
			return peers
		}
		// Fall back to allowing any destination if discovery failed
		// Port restriction still provides security
		return []networkingv1.NetworkPolicyPeer{
			{
				IPBlock: &networkingv1.IPBlock{CIDR: "0.0.0.0/0"},
			},
		}
	case netPolicyOAuthServer:
		// Allow egress to OpenShift OAuth server for token exchange
		// The OAuth server can be accessed via route (external) or service (internal)
		// so we allow both ipBlock and namespaceSelector
		return []networkingv1.NetworkPolicyPeer{
			{
				IPBlock: &networkingv1.IPBlock{CIDR: "0.0.0.0/0"},
			},
			{
				NamespaceSelector: &metav1.LabelSelector{},
			},
		}
	default:
		return []networkingv1.NetworkPolicyPeer{
			{
				PodSelector: &metav1.LabelSelector{
					MatchLabels: manifestutils.ComponentLabels(name, tempo.Name),
				},
			},
		}
	}
}

// networkRelations define connections: from -> to using NetworkPolicyPort.
type networkRelations = map[string]map[string][]networkingv1.NetworkPolicyPort

const (
	netPolicys3Storage         = "s3"
	netPolicyOtelTargets       = "otel"
	netPolicyClusterComponents = "cluster"
	netPolicyKubeAPIServer     = "kube-apiserver"
	netPolicyOAuthServer       = "oauth-server"
)

func componentRelations(params manifestutils.Params) networkRelations {
	tempo := params.Tempo
	var (
		s3Conn   = extractStoragePorts(params.StorageParams)
		grpcConn = []networkingv1.NetworkPolicyPort{
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(manifestutils.PortGRPCServer)),
			},
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(manifestutils.PortHTTPServer)),
			},
		}
		otelExport = []networkingv1.NetworkPolicyPort{
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(manifestutils.PortOtlpHttp)),
			},
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(manifestutils.PortOtlpGrpcServer)),
			},
		}
		// Use discovered Kubernetes API server ports, or fall back to default 6443
		kubeAPIServer = params.KubeAPIServer.Ports
		oauthServer   = []networkingv1.NetworkPolicyPort{
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(443)),
			},
		}
	)

	fromTo := map[string]map[string][]networkingv1.NetworkPolicyPort{
		manifestutils.GatewayComponentName:       {},
		manifestutils.DistributorComponentName:   {},
		manifestutils.IngesterComponentName:      {},
		manifestutils.QueryFrontendComponentName: {},
		manifestutils.QuerierComponentName:       {},
		manifestutils.CompactorComponentName:     {},
	}

	// Assign storage connections to components that need direct storage access
	fromTo[manifestutils.IngesterComponentName][netPolicys3Storage] = s3Conn
	fromTo[manifestutils.QuerierComponentName][netPolicys3Storage] = s3Conn
	fromTo[manifestutils.QueryFrontendComponentName][netPolicys3Storage] = s3Conn
	fromTo[manifestutils.CompactorComponentName][netPolicys3Storage] = s3Conn

	// Distributor sends traces to Ingesters
	fromTo[manifestutils.DistributorComponentName][manifestutils.IngesterComponentName] = grpcConn

	// Querier connects to ingesters
	fromTo[manifestutils.QuerierComponentName][manifestutils.IngesterComponentName] = grpcConn

	// Bidirectional query-frontend <-> querier communication
	// Querier connects to query-frontend on port 9095 (frontend worker registration)
	fromTo[manifestutils.QuerierComponentName][manifestutils.QueryFrontendComponentName] = grpcConn
	// Query-frontend connects to querier (sending queries)
	fromTo[manifestutils.QueryFrontendComponentName][manifestutils.QuerierComponentName] = grpcConn

	if tempo.Spec.Observability.Tracing.OTLPHttpEndpoint != "" {
		fromTo[manifestutils.DistributorComponentName][netPolicyOtelTargets] = otelExport
		fromTo[manifestutils.IngesterComponentName][netPolicyOtelTargets] = otelExport
		fromTo[manifestutils.QuerierComponentName][netPolicyOtelTargets] = otelExport
		fromTo[manifestutils.QueryFrontendComponentName][netPolicyOtelTargets] = otelExport
		fromTo[manifestutils.CompactorComponentName][netPolicyOtelTargets] = otelExport

		// Gateway telemetry export when gateway is enabled
		if tempo.Spec.Template.Gateway.Enabled {
			fromTo[manifestutils.GatewayComponentName][netPolicyOtelTargets] = otelExport
		}
	}

	fromTo[netPolicyClusterComponents] = map[string][]networkingv1.NetworkPolicyPort{}
	if tempo.Spec.Template.Gateway.Enabled {
		// Allow external access to Gateway HTTP and gRPC ports
		fromTo[netPolicyClusterComponents][manifestutils.GatewayComponentName] = []networkingv1.NetworkPolicyPort{
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(manifestutils.GatewayPortHTTPServer)),
			},
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(manifestutils.GatewayPortGRPCServer)),
			},
		}

		// Gateway sends OTLP to Distributor
		fromTo[manifestutils.GatewayComponentName][manifestutils.DistributorComponentName] = []networkingv1.NetworkPolicyPort{
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(manifestutils.PortOtlpGrpcServer)),
			},
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(manifestutils.PortOtlpHttp)),
			},
		}

		// Gateway queries via Query Frontend
		fromTo[manifestutils.GatewayComponentName][manifestutils.QueryFrontendComponentName] = grpcConn

		// Gateway needs to access Kubernetes API server for TokenReview/SubjectAccessReview
		// when using OpenShift RBAC mode for multi-tenancy
		fromTo[manifestutils.GatewayComponentName][netPolicyKubeAPIServer] = kubeAPIServer

		// Gateway needs to access OpenShift OAuth server for token exchange
		// when using OpenShift authentication mode
		fromTo[manifestutils.GatewayComponentName][netPolicyOAuthServer] = oauthServer

		// Gateway needs to access Jaeger Query UI ports when JaegerQuery is enabled
		if tempo.Spec.Template.QueryFrontend.JaegerQuery.Enabled {
			fromTo[manifestutils.GatewayComponentName][manifestutils.QueryFrontendComponentName] = append(
				fromTo[manifestutils.GatewayComponentName][manifestutils.QueryFrontendComponentName],
				networkingv1.NetworkPolicyPort{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     ptr.To(intstr.FromInt(manifestutils.PortJaegerUI)),
				},
				networkingv1.NetworkPolicyPort{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     ptr.To(intstr.FromInt(manifestutils.PortJaegerMetrics)),
				},
			)
		}
	} else {
		// Allow external access to Distributor receiver ports (when gateway is disabled)
		fromTo[netPolicyClusterComponents][manifestutils.DistributorComponentName] = []networkingv1.NetworkPolicyPort{
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(manifestutils.PortOtlpGrpcServer)),
			},
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(manifestutils.PortOtlpHttp)),
			},
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(manifestutils.PortJaegerThriftHTTP)),
			},
			{
				Protocol: ptr.To(corev1.ProtocolUDP),
				Port:     ptr.To(intstr.FromInt(manifestutils.PortJaegerThriftCompact)),
			},
			{
				Protocol: ptr.To(corev1.ProtocolUDP),
				Port:     ptr.To(intstr.FromInt(manifestutils.PortJaegerThriftBinary)),
			},
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(manifestutils.PortJaegerGrpc)),
			},
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(manifestutils.PortZipkin)),
			},
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(manifestutils.PortHTTPServer)),
			},
		}

		// Allow external access to Query Frontend for querying (when gateway is disabled)
		fromTo[netPolicyClusterComponents][manifestutils.QueryFrontendComponentName] = grpcConn
	}

	if tempo.Spec.Template.QueryFrontend.JaegerQuery.Enabled {
		fromTo[netPolicyClusterComponents][manifestutils.QueryFrontendComponentName] = append(
			fromTo[netPolicyClusterComponents][manifestutils.QueryFrontendComponentName],
			networkingv1.NetworkPolicyPort{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(manifestutils.PortJaegerUI)),
			},
			networkingv1.NetworkPolicyPort{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(manifestutils.PortJaegerMetrics)),
			},
		)

		// Add oauth-proxy port when Jaeger Query authentication is enabled (single-tenant auth)
		if tempo.Spec.Template.QueryFrontend.JaegerQuery.Authentication != nil &&
			tempo.Spec.Template.QueryFrontend.JaegerQuery.Authentication.Enabled {
			fromTo[netPolicyClusterComponents][manifestutils.QueryFrontendComponentName] = append(
				fromTo[netPolicyClusterComponents][manifestutils.QueryFrontendComponentName],
				networkingv1.NetworkPolicyPort{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     ptr.To(intstr.FromInt(manifestutils.OAuthProxyPort)),
				},
			)
		}
	}
	return fromTo
}

func reverseRelations(rels map[string]map[string][]networkingv1.NetworkPolicyPort) map[string]map[string][]networkingv1.NetworkPolicyPort {
	reverse := map[string]map[string][]networkingv1.NetworkPolicyPort{}

	for from, targets := range rels {
		for to, ports := range targets {
			if _, ok := reverse[to]; !ok {
				reverse[to] = map[string][]networkingv1.NetworkPolicyPort{}
			}
			reverse[to][from] = append(reverse[to][from], ports...)
		}
	}

	return reverse
}

// extractStoragePorts extracts the storage port for network policies.
// It handles S3 (with custom endpoints), Azure Storage, and GCS.
// Azure and GCS always use HTTPS (port 443).
// S3 can have custom endpoints with custom ports, or defaults to 443 (HTTPS) or 80 (HTTP).
func extractStoragePorts(storageParams manifestutils.StorageParams) []networkingv1.NetworkPolicyPort {
	if storageParams.S3 != nil {
		port := 0
		endpoint := storageParams.S3.Endpoint

		if storageParams.CredentialMode == "static" && endpoint != "" {
			// Endpoint format is "hostname:port" (scheme already stripped)
			if colonIdx := strings.LastIndexByte(endpoint, ':'); colonIdx != -1 {
				portStr := endpoint[colonIdx+1:]
				// Check if it's a valid port (not an IPv6 address)
				if p, err := strconv.Atoi(portStr); err == nil && p > 0 && p <= 65535 {
					port = p
				}
			}
		}

		if port == 0 {
			if storageParams.S3.Insecure {
				port = 80
			} else {
				port = 443
			}
		}

		return []networkingv1.NetworkPolicyPort{
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(port)),
			},
		}
	}

	// Azure Storage and GCS always use HTTPS (port 443)
	if storageParams.AzureStorage != nil || storageParams.GCS != nil {
		return []networkingv1.NetworkPolicyPort{
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(443)),
			},
		}
	}

	// No storage configured
	return []networkingv1.NetworkPolicyPort{}
}

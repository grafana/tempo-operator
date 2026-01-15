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

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
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
		for _, conn := range ports {
			peer := policyPeerFor(target, tempo)
			np.Spec.Egress = append(np.Spec.Egress, networkingv1.NetworkPolicyEgressRule{
				Ports: []networkingv1.NetworkPolicyPort{conn},
				To:    []networkingv1.NetworkPolicyPeer{peer},
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
			peer := policyPeerFor(target, tempo)
			np.Spec.Ingress = append(np.Spec.Ingress, networkingv1.NetworkPolicyIngressRule{
				Ports: conn,
				From:  []networkingv1.NetworkPolicyPeer{peer},
			})
		}
	}

	if len(np.Spec.Ingress) > 0 {
		np.Spec.PolicyTypes = append(np.Spec.PolicyTypes, networkingv1.PolicyTypeIngress)
	}

	return np
}

func policyPeerFor(name string, tempo v1alpha1.TempoStack) networkingv1.NetworkPolicyPeer {
	switch name {
	case netPolicyOtelTargets:
		return networkingv1.NetworkPolicyPeer{
			IPBlock: &networkingv1.IPBlock{CIDR: "0.0.0.0/0"},
		}
	case netPolicys3Storage:
		// Allow egress to any namespace and any pod for S3 storage access
		// This is necessary for cross-namespace access to object storage like MinIO
		return networkingv1.NetworkPolicyPeer{
			NamespaceSelector: &metav1.LabelSelector{},
			PodSelector:       &metav1.LabelSelector{},
		}
	case netPolicyClusterComponents:
		return networkingv1.NetworkPolicyPeer{
			NamespaceSelector: &metav1.LabelSelector{},
		}
	default:
		return networkingv1.NetworkPolicyPeer{
			PodSelector: &metav1.LabelSelector{
				MatchLabels: manifestutils.ComponentLabels(name, tempo.Name),
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
	fromTo[manifestutils.CompactorComponentName][netPolicys3Storage] = s3Conn

	// Distributor sends traces to Ingesters
	fromTo[manifestutils.DistributorComponentName][manifestutils.IngesterComponentName] = grpcConn

	// Querier connects to query-frontend and ingesters
	fromTo[manifestutils.QuerierComponentName][manifestutils.QueryFrontendComponentName] = grpcConn
	fromTo[manifestutils.QuerierComponentName][manifestutils.IngesterComponentName] = grpcConn

	// Query Frontend connects to Queriers
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

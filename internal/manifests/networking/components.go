package networking

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

func generatePolicyFor(tempo v1alpha1.TempoStack, componentName string) *networkingv1.NetworkPolicy {
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

	rels := componentRelations(tempo)[componentName]

	for target, ports := range rels {
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

	reverse := reverseRelations(componentRelations(tempo))
	for source, ports := range reverse {
		for target, conn := range ports {
			if target != componentName {
				continue
			}
			peer := policyPeerFor(source, tempo)
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
	case netPolicys3Storage, netPolicyOtelTargets:
		return networkingv1.NetworkPolicyPeer{
			IPBlock: &networkingv1.IPBlock{CIDR: "0.0.0.0/0"},
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

func componentRelations(tempo v1alpha1.TempoStack) networkRelations {
	var (
		s3Conn = []networkingv1.NetworkPolicyPort{
			{ // TODO: get this from secret?
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(443)),
			},
			{ // TODO: get this from secret?
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(9000)),
			},
		}
		tempoGrpcConn = networkingv1.NetworkPolicyPort{
			Protocol: ptr.To(corev1.ProtocolTCP),
			Port:     ptr.To(intstr.FromString(manifestutils.GrpcPortName)),
		}
		otelHttpConn = networkingv1.NetworkPolicyPort{
			Protocol: ptr.To(corev1.ProtocolTCP),
			Port:     ptr.To(intstr.FromInt(4318)),
		}
		otelGrpcConn = networkingv1.NetworkPolicyPort{
			Protocol: ptr.To(corev1.ProtocolTCP),
			Port:     ptr.To(intstr.FromInt(4317)),
		}
	)
	clusterIngress := map[string][]networkingv1.NetworkPolicyPort{}
	if tempo.Spec.Template.Gateway.Enabled { // TODO: add cluster -> gateway access
		clusterIngress[manifestutils.GatewayComponentName] = []networkingv1.NetworkPolicyPort{}
	}

	if tempo.Spec.Template.QueryFrontend.JaegerQuery.Enabled { // TODO: add cluster -> jaegerQuery access
		clusterIngress[manifestutils.JaegerFrontendComponentName] = []networkingv1.NetworkPolicyPort{}
	}
	return map[string]map[string][]networkingv1.NetworkPolicyPort{
		netPolicyClusterComponents: clusterIngress,
		manifestutils.DistributorComponentName: {
			manifestutils.IngesterComponentName: {
				tempoGrpcConn,
			},
			netPolicyOtelTargets: {
				otelGrpcConn,
				otelHttpConn,
			},
		},
		manifestutils.IngesterComponentName: {
			netPolicys3Storage: s3Conn,
		},
		manifestutils.QuerierComponentName: {
			manifestutils.IngesterComponentName: {
				tempoGrpcConn,
			},
			netPolicys3Storage: s3Conn,
		},
		manifestutils.QueryFrontendComponentName: {
			manifestutils.QuerierComponentName: {
				tempoGrpcConn,
			},
		},
		manifestutils.CompactorComponentName: {
			netPolicys3Storage: s3Conn,
		},
	}
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

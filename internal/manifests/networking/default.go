package networking

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

func GenerateOperandPolicies(tempo v1alpha1.TempoStack) []client.Object {
	if !tempo.Spec.Networking.Enabled {
		return nil
	}
	policies := []client.Object{
		policyDenyAll(tempo),
		policyIngressToMetrics(tempo),
		policyEgressAllowDNS(tempo),
	}

	policies = append(policies, generatePolicyFor(tempo, manifestutils.DistributorComponentName))
	policies = append(policies, generatePolicyFor(tempo, manifestutils.IngesterComponentName))
	policies = append(policies, generatePolicyFor(tempo, manifestutils.CompactorComponentName))
	policies = append(policies, generatePolicyFor(tempo, manifestutils.QuerierComponentName))
	policies = append(policies, generatePolicyFor(tempo, manifestutils.QueryFrontendComponentName))

	return policies
}

func policyDenyAll(tempo v1alpha1.TempoStack) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-deny", naming.Name("", tempo.Name)),
			Namespace: tempo.Namespace,
			Labels:    manifestutils.CommonLabels(tempo.Name),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: manifestutils.CommonLabels(tempo.Name),
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
		},
	}
}

func policyIngressToMetrics(tempo v1alpha1.TempoStack) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-ingress-to-metrics", naming.Name("", tempo.Name)),
			Namespace: tempo.Namespace,
			Labels:    manifestutils.CommonLabels(tempo.Name),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: manifestutils.CommonLabels(tempo.Name),
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector:       &metav1.LabelSelector{},
							NamespaceSelector: &metav1.LabelSelector{},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: ptr.To(corev1.ProtocolTCP),
							Port:     ptr.To(intstr.FromString("metrics")),
						},
					},
				},
			},
		},
	}
}

func policyEgressAllowDNS(tempo v1alpha1.TempoStack) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-allow-dns", naming.Name("", tempo.Name)),
			Namespace: tempo.Namespace,
			Labels:    manifestutils.CommonLabels(tempo.Name),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: manifestutils.CommonLabels(tempo.Name),
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: ptr.To(corev1.ProtocolTCP),
							Port:     ptr.To(intstr.FromInt(53)),
						},
						{
							Protocol: ptr.To(corev1.ProtocolUDP),
							Port:     ptr.To(intstr.FromInt(53)),
						},
					},
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name:": "openshift-dns",
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"dns.operator.openshift.io/daemonset-dns": "default",
								},
							},
						},
					},
				},
			},
		},
	}
}

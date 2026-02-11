package networkpolicies

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

func policyAPIServer(instanceName, namespace string, apiServerInfo manifestutils.KubeAPIServerInfo) *networkingv1.NetworkPolicy {
	egressRule := networkingv1.NetworkPolicyEgressRule{
		Ports: apiServerInfo.Ports,
	}

	// Add IP-based peers if specific IPs are available, otherwise allow all destinations
	if len(apiServerInfo.IPs) > 0 {
		for _, ip := range apiServerInfo.IPs {
			egressRule.To = append(egressRule.To, networkingv1.NetworkPolicyPeer{
				IPBlock: &networkingv1.IPBlock{CIDR: ip + "/32"},
			})
		}
	}
	// If no specific IPs, To field remains nil which allows all destinations (port restriction still applies)

	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-egress-to-apiserver", naming.Name("", instanceName)),
			Namespace: namespace,
			Labels:    manifestutils.CommonOperatorLabels(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: manifestutils.CommonOperatorLabels(),
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{egressRule},
		},
	}
}

func policyWebhook(instanceName, namespace string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-ingress-webhook", naming.Name("", instanceName)),
			Namespace: namespace,
			Labels:    manifestutils.CommonOperatorLabels(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: manifestutils.CommonOperatorLabels(),
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: ptr.To(corev1.ProtocolTCP),
							Port:     ptr.To(intstr.FromInt(9443)),
						},
					},
				},
			},
		},
	}
}

func policyDenyAll(instanceName, namespace string, labels map[string]string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-deny-all", naming.Name("", instanceName)),
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
		},
	}
}

func policyIngressToMetrics(instanceName, namespace string, labels map[string]string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-ingress-to-metrics", naming.Name("", instanceName)),
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: labels,
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
							Port:     ptr.To(intstr.FromInt(8443)),
						},
					},
				},
			},
		},
	}
}

func policyTempoGossip(instanceName, namespace string, labels map[string]string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-gossip", naming.Name("", instanceName)),
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: ptr.To(corev1.ProtocolTCP),
							Port:     ptr.To(intstr.FromInt(manifestutils.PortMemberlist)),
						},
						{
							Protocol: ptr.To(corev1.ProtocolTCP),
							Port:     ptr.To(intstr.FromInt(3200)),
						},
						{
							Protocol: ptr.To(corev1.ProtocolTCP),
							Port:     ptr.To(intstr.FromInt(3101)),
						},
					},
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: labels,
							},
						},
					},
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: ptr.To(corev1.ProtocolTCP),
							Port:     ptr.To(intstr.FromInt(manifestutils.PortMemberlist)),
						},
						{
							Protocol: ptr.To(corev1.ProtocolTCP),
							Port:     ptr.To(intstr.FromInt(3200)),
						},
						{
							Protocol: ptr.To(corev1.ProtocolTCP),
							Port:     ptr.To(intstr.FromInt(3101)),
						},
					},
					To: []networkingv1.NetworkPolicyPeer{
						{
							// NamespaceSelector: &metav1.LabelSelector{
							// 	MatchLabels: map[string]string{
							// 		"kubernetes.io/metadata.name": "openshift-dns",
							// 	},
							// },
							PodSelector: &metav1.LabelSelector{
								MatchLabels: labels,
							},
						},
					},
				},
			},
		},
	}
}

func policyIngressToOperandMetrics(instanceName, namespace string, labels map[string]string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-ingress-to-operand-metrics", naming.Name("", instanceName)),
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      "openshift.io/cluster-monitoring",
										Operator: metav1.LabelSelectorOpExists,
									},
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: ptr.To(corev1.ProtocolTCP),
							Port:     ptr.To(intstr.FromInt(manifestutils.PortHTTPServer)),
						},
					},
				},
			},
		},
	}
}

func policyEgressAllowDNS(instanceName, namespace string, labels map[string]string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-allow-dns", naming.Name("", instanceName)),
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					// Standard Kubernetes DNS on port 53
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
									"kubernetes.io/metadata.name": "kube-system",
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"k8s-app": "kube-dns",
								},
							},
						},
					},
				},
			},
		},
	}
}

func policyEgressAllowDNSOpenShift(instanceName, namespace string, labels map[string]string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-allow-dns-openshift", naming.Name("", instanceName)),
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					// OpenShift DNS on port 5353
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: ptr.To(corev1.ProtocolTCP),
							Port:     ptr.To(intstr.FromInt(5353)),
						},
						{
							Protocol: ptr.To(corev1.ProtocolUDP),
							Port:     ptr.To(intstr.FromInt(5353)),
						},
					},
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": "openshift-dns",
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

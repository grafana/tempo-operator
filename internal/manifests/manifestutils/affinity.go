package manifestutils

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ConfigureAffinity returns the affinity of a Tempo component.
//
// By default two pods of a component prefer to be scheduled onto different nodes and failure
// domains. A component may replace that default with its own anti affinity, for example to require
// rather than prefer a spread across nodes, via spec.template.<component>.podAntiAffinity.
//
// The behavior follows the Grafana Loki Operator, which exposes the same field per component.
func ConfigureAffinity(labels labels.Set, podAntiAffinity *corev1.PodAntiAffinity) *corev1.Affinity {
	affinity := DefaultAffinity(labels)
	if podAntiAffinity != nil {
		affinity.PodAntiAffinity = podAntiAffinity
	}
	return affinity
}

// DefaultAffinity returns the default affinity for Tempo components.
// It defines that two pods with the same labels (i.e. same component)
// should not be scheduled on the same node or failure domain.
func DefaultAffinity(labels labels.Set) *corev1.Affinity {
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: labels,
						},
						TopologyKey: corev1.LabelHostname,
					},
				},
				{
					Weight: 75,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: labels,
						},
						TopologyKey: "failure-domain.beta.kubernetes.io/zone",
					},
				},
			},
		},
	}
}

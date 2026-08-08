package manifestutils

import (
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

// NewPodDisruptionBudget returns a PodDisruptionBudget for the given TempoStack
// component. It uses maxUnavailable: 1 so that voluntary disruptions (e.g. node
// drains during upgrades) can only take down a single replica of the component
// at a time, keeping the rest available.
func NewPodDisruptionBudget(tempo v1alpha1.TempoStack, component string) *policyv1.PodDisruptionBudget {
	labels := ComponentLabels(component, tempo.Name)
	return &policyv1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodDisruptionBudget",
			APIVersion: policyv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(component, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector:       &metav1.LabelSelector{MatchLabels: labels},
			MaxUnavailable: ptr.To(intstr.FromInt32(1)),
		},
	}
}

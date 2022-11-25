package serviceaccount

import (
	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	componentName = "serviceaccount"
)

// ServiceAccountName returns the name of the custom service account (if specified in the CR) or the default tempo service account to use.
func ServiceAccountName(tempo v1alpha1.Microservices) string {
	if tempo.Spec.ServiceAccount != "" {
		return tempo.Spec.ServiceAccount
	}
	return manifestutils.Name(componentName, tempo.Name)
}

// BuildServiceAccount creates a Kubernetes service account for tempo.
func BuildDefaultServiceAccount(tempo v1alpha1.Microservices) *corev1.ServiceAccount {
	labels := manifestutils.ComponentLabels(componentName, tempo.Name)
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      manifestutils.Name(componentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
	}
}

package serviceaccount

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	"github.com/os-observability/tempo-operator/internal/manifests/naming"
)

const (
	componentName = "serviceaccount"
)

// BuildDefaultServiceAccount creates a Kubernetes service account for tempo.
func BuildDefaultServiceAccount(tempo v1alpha1.Microservices) *corev1.ServiceAccount {
	labels := manifestutils.ComponentLabels(componentName, tempo.Name)
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.DefaultServiceAccountName(tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
	}
}

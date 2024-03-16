package monolithic

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/gateway"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

const (
	componentName = "serviceaccount"
)

// BuildServiceAccount creates a Kubernetes service account for Tempo.
func BuildServiceAccount(opts Options) *corev1.ServiceAccount {
	tempo := opts.Tempo
	labels := ComponentLabels(componentName, tempo.Name)
	var annotations map[string]string
	if tempo.Spec.Multitenancy.IsGatewayEnabled() && tempo.Spec.Multitenancy.Mode == v1alpha1.ModeOpenShift {
		annotations = gateway.BuildServiceAccountAnnotations(tempo.Spec.Multitenancy.TenantsSpec, naming.Name("jaegerui", tempo.Name))
	}

	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.DefaultServiceAccountName(tempo.Name),
			Namespace:   tempo.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
	}
}

package status

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

// StatusClient defines a interface for fetching status information.
type StatusClient interface {
	GetPodsComponent(ctx context.Context, componentName string, stack v1alpha1.Microservices) (*corev1.PodList, error)
	PatchStatus(ctx context.Context, changed, original *v1alpha1.Microservices) error
}

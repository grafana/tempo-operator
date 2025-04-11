package serviceaccount

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

const (
	componentName = "serviceaccount"
)

// BuildDefaultServiceAccount creates a Kubernetes service account for tempo.
func BuildDefaultServiceAccount(params manifestutils.Params) *corev1.ServiceAccount {
	labels := manifestutils.ComponentLabels(componentName, params.Tempo.Name)
	var annotations map[string]string
	if params.StorageParams.S3 != nil && params.StorageParams.S3.ShortLived != nil {
		annotations = manifestutils.S3AWSSTSAnnotations(*params.StorageParams.S3.ShortLived)
	}

	if params.StorageParams.GCS != nil && params.StorageParams.GCS.ShortLived != nil {
		annotations = manifestutils.GCSShortLiveTokenAnnotation(*params.StorageParams.GCS.ShortLived)
	}

	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.DefaultServiceAccountName(params.Tempo.Name),
			Namespace:   params.Tempo.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
	}
}

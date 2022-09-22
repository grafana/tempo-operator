package ingester

import (
	"github.com/os-observability/tempo-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func BuildIngester(tempo v1alpha1.Microservices) []client.Object {
	return []client.Object{deployment(tempo)}
}

func deployment(tempo v1alpha1.Microservices) *v1.Deployment {
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: v1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "",
							Image: "",
						},
					},
				},
			},
		},
	}
}

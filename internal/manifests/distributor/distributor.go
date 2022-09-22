package distributor

import (
	"github.com/os-observability/tempo-operator/api/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func BuildDistributor(params v1alpha1.Microservices) []client.Object {
	return []client.Object{deployment(params)}
}

func deployment(tempo v1alpha1.Microservices) *v1.StatefulSet {
	return &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: v1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  manifestutils.Name(tempo.Name, "deployment"),
							Image: "docker.io/grafana/tempo:1.4.1",
							Args:  []string{"-target=distributor", "-config.file=/conf/tempo.yaml"},
						},
					},
				},
			},
		},
	}
}

package distributor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/os-observability/tempo-operator/api/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildDistributor(t *testing.T) {
	objects := BuildDistributor(v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "project1",
		},
	})

	labels := manifestutils.ComponentLabels("distributor", "test")
	assert.Equal(t, 2, len(objects))
	assert.Equal(t, &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-test-distributor",
			Namespace: "project1",
			Labels:    labels,
		},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "tempo",
							Image: "docker.io/grafana/tempo:1.5.0",
							Args:  []string{"-target=distributor", "-config.file=/conf/tempo.yaml"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      configVolumeName,
									MountPath: "/conf",
									ReadOnly:  true,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          otlpGrpcPortName,
									ContainerPort: otlpGrpcPort,
									Protocol:      corev1.ProtocolTCP,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: configVolumeName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "tempo-test",
									},
								},
							},
						},
					},
				},
			},
		},
	}, objects[0])
	assert.Equal(t, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-test-distributor",
			Namespace: "project1",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       otlpGrpcPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       otlpGrpcPort,
					TargetPort: intstr.FromString(otlpGrpcPortName),
				},
			},
			Selector: labels,
		},
	}, objects[1])
}

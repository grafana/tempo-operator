package ingester

import (
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/os-observability/tempo-operator/api/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
)

const (
	configVolumeName = "tempo-conf"
	componentName    = "ingester"
	portGRPCServer   = 9095
	portHTTPServer   = 3100
	portMemberlist   = 7946
)

// BuildIngester creates distributor objects.
func BuildIngester(tempo v1alpha1.Microservices) []client.Object {
	return []client.Object{deployment(tempo), service(tempo)}
}

func deployment(tempo v1alpha1.Microservices) *v1.StatefulSet {
	labels := manifestutils.ComponentLabels("ingester", tempo.Name)
	return &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      manifestutils.Name("ingester", tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: v1.StatefulSetSpec{
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
							Args:  []string{"-target=ingester", "-config.file=/conf/tempo.yaml"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      configVolumeName,
									MountPath: "/conf",
									ReadOnly:  true,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "memberlist",
									ContainerPort: portMemberlist,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "http",
									ContainerPort: portHTTPServer,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "grpc",
									ContainerPort: portGRPCServer,
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
										Name: manifestutils.Name("", tempo.Name),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func service(tempo v1alpha1.Microservices) *corev1.Service {
	labels := manifestutils.ComponentLabels(componentName, tempo.Name)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      manifestutils.Name(componentName, tempo.Name),
			Namespace: tempo.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       portHTTPServer,
					TargetPort: intstr.FromString("http"),
				},
				{
					Name:       "grpc",
					Protocol:   corev1.ProtocolTCP,
					Port:       portGRPCServer,
					TargetPort: intstr.FromString("grpc"),
				},
			},
			Selector: labels,
		},
	}
}

package distributor

import (
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	"github.com/os-observability/tempo-operator/internal/manifests/memberlist"
	"github.com/os-observability/tempo-operator/internal/manifests/serviceaccount"
)

const (
	configVolumeName = "tempo-conf"
	componentName    = "distributor"
	otlpGrpcPortName = "otlp-grpc"
	otlpGrpcPort     = 4317
)

// BuildDistributor creates distributor objects.
func BuildDistributor(tempo v1alpha1.Microservices) []client.Object {
	return []client.Object{deployment(tempo), service(tempo)}
}

func deployment(tempo v1alpha1.Microservices) *v1.Deployment {
	labels := manifestutils.ComponentLabels(componentName, tempo.Name)
	cfg := &v1alpha1.TempoComponentSpec{}
	if userCfg := tempo.Spec.Components.Distributor; userCfg != nil {
		cfg = userCfg
	}

	return &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      manifestutils.Name(componentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: k8slabels.Merge(labels, memberlist.GossipSelector),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceaccount.ServiceAccountName(tempo),
					NodeSelector:       cfg.NodeSelector,
					Tolerations:        cfg.Tolerations,
					Containers: []corev1.Container{
						{
							Name:  "tempo",
							Image: tempo.Spec.Images.Tempo,
							Args:  []string{"-target=distributor", "-config.file=/conf/tempo.yaml"},
							Ports: []corev1.ContainerPort{
								{
									Name:          otlpGrpcPortName,
									ContainerPort: otlpGrpcPort,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "http-memberlist",
									ContainerPort: memberlist.PortMemberlist,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      configVolumeName,
									MountPath: "/conf",
									ReadOnly:  true,
								},
							},
							Resources: manifestutils.Resources(tempo, componentName),
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
			Labels:    labels,
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
	}
}

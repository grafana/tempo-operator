package ingester

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
	"github.com/os-observability/tempo-operator/internal/manifests/naming"
)

const (
	configVolumeName = "tempo-conf"
	dataVolumeName   = "data"
	componentName    = "ingester"
	portGRPCServer   = 9095
	portHTTPServer   = 3100
)

// BuildIngester creates distributor objects.
func BuildIngester(tempo v1alpha1.Microservices) ([]client.Object, error) {
	ss, err := statefulSet(tempo)
	if err != nil {
		return nil, err
	}

	return []client.Object{ss, service(tempo)}, nil
}

func statefulSet(tempo v1alpha1.Microservices) (*v1.StatefulSet, error) {
	labels := manifestutils.ComponentLabels(componentName, tempo.Name)
	filesystem := corev1.PersistentVolumeFilesystem
	cfg := &v1alpha1.TempoComponentSpec{}
	if userCfg := tempo.Spec.Components.Ingester; userCfg != nil {
		cfg = userCfg
	}

	ss := &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(componentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: v1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: k8slabels.Merge(labels, memberlist.GossipSelector),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: tempo.Spec.ServiceAccount,
					NodeSelector:       cfg.NodeSelector,
					Tolerations:        cfg.Tolerations,
					Containers: []corev1.Container{
						{
							Name:  "tempo",
							Image: tempo.Spec.Images.Tempo,
							Args:  []string{"-target=ingester", "-config.file=/conf/tempo.yaml"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      configVolumeName,
									MountPath: "/conf",
									ReadOnly:  true,
								},
								{
									Name:      dataVolumeName,
									MountPath: "/var/tempo",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http-memberlist",
									ContainerPort: memberlist.PortMemberlist,
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
							Resources: manifestutils.Resources(tempo, componentName),
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: configVolumeName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: naming.Name("", tempo.Name),
									},
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: dataVolumeName,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: tempo.Spec.StorageSize,
							},
						},
						StorageClassName: tempo.Spec.StorageClassName,
						VolumeMode:       &filesystem,
					},
				},
			},
		},
	}
	err := manifestutils.ConfigureStorage(tempo, &ss.Spec.Template.Spec)
	if err != nil {
		return nil, err
	}
	return ss, nil
}

func service(tempo v1alpha1.Microservices) *corev1.Service {
	labels := manifestutils.ComponentLabels(componentName, tempo.Name)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(componentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
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

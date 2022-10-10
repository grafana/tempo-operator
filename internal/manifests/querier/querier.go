package querier

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
)

const (
	configVolumeName = "tempo-conf"
	componentName    = "querier"
	httpPortName     = "http"
	grpcPortName     = "grpc"
	portHTTPServer   = 3100
	portGRPCServer   = 9095
)

// BuildQuerier creates querier objects.
func BuildQuerier(tempo v1alpha1.Microservices) ([]client.Object, error) {
	d, err := deployment(tempo)
	if err != nil {
		return nil, err
	}

	return []client.Object{d, service(tempo)}, nil
}

func deployment(tempo v1alpha1.Microservices) (*v1.Deployment, error) {
	labels := manifestutils.ComponentLabels(componentName, tempo.Name)
	d := &v1.Deployment{
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
					Containers: []corev1.Container{
						{
							Name:  "tempo",
							Image: "docker.io/grafana/tempo:1.5.0",
							Args:  []string{"-target=querier", "-config.file=/conf/tempo.yaml", "-mem-ballast-size-mbs=1024"},
							Ports: []corev1.ContainerPort{
								{
									Name:          httpPortName,
									ContainerPort: portHTTPServer,
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

	err := manifestutils.ConfigureStorage(tempo, &d.Spec.Template.Spec)
	if err != nil {
		return nil, err
	}
	return d, nil
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
					Name:       httpPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       memberlist.PortMemberlist,
					TargetPort: intstr.FromInt(memberlist.PortMemberlist),
				},
				{
					Name:       grpcPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       portGRPCServer,
					TargetPort: intstr.FromString("grpc"),
				},
			},
			Selector: labels,
		},
	}
}

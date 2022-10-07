package queryfrontend

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
	configVolumeName           = "tempo-conf"
	componentName              = "query-frontend"
	grpcPortName               = "grpc"
	grpclbPortName             = "grpclb"
	httpPortName               = "http"
	jaegerMetricsPortName      = "jaeger-metrics"
	jaegerUIPortName           = "jaeger-ui"
	tempoQueryJaegerUiPortName = "tempo-query-jaeger-ui"
	tempoQueryMetricsPortName  = "tempo-query-metrics"
	portHTTPServer             = 3100
	portGRPCServer             = 9095
	portGRPCLBServer           = 9096
	portJaegerUI               = 16686
	portQueryMetrics           = 16687
)

// BuildQueryFrontend creates the query-frontend objects.
func BuildQueryFrontend(tempo v1alpha1.Microservices) []client.Object {
	theDeployment := deployment(tempo)
	queryFrontEndServices := services(tempo)

	return []client.Object{theDeployment, queryFrontEndServices[0], queryFrontEndServices[1]}
}

func deployment(tempo v1alpha1.Microservices) *v1.Deployment {
	labels := manifestutils.ComponentLabels(componentName, tempo.Name)

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
					// TODO do we need to set any of this affinity stuff???
					Containers: []corev1.Container{
						{
							Name:  "query-frontend",
							Image: "docker.io/grafana/tempo:1.5.0",
							Args:  []string{"-target=query-frontend", "-config.file=/conf/tempo.yaml", "-mem-ballast-size-mbs=1024"},
							Ports: []corev1.ContainerPort{
								{
									Name:          httpPortName,
									ContainerPort: portHTTPServer,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          grpcPortName,
									ContainerPort: portGRPCServer,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							// TODO do we need to set resources
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      configVolumeName,
									MountPath: "/conf",
									ReadOnly:  true,
								},
							},
						},
						{
							Name:  "tempo",
							Image: "docker.io/grafana/tempo:1.5.0",
							Args: []string{"--query.base-path=/",
								"--grpc-storage-plugin.configuration-file=/conf/tempo-query.yaml",
								"--query.bearer-token-propagation=true"},
							Ports: []corev1.ContainerPort{
								{
									Name:          jaegerUIPortName,
									ContainerPort: portJaegerUI,
								},
								{
									Name:          jaegerMetricsPortName,
									ContainerPort: portQueryMetrics,
								},
							},
							// TODO do we need to define Resources?
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
						// TODO our tempo deployment has two volumes here named "data-querier-frontend" and "data-query"
						//  both have "emptyDir" set to null.
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

func services(tempo v1alpha1.Microservices) []*corev1.Service {
	labels := manifestutils.ComponentLabels(componentName, tempo.Name)

	frontEndService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      manifestutils.Name(componentName, tempo.Name),
			Namespace: tempo.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       httpPortName,
					Port:       portHTTPServer,
					TargetPort: intstr.FromInt(portHTTPServer),
				},
				{
					Name:       grpcPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       portGRPCServer,
					TargetPort: intstr.FromInt(portGRPCServer),
				},
				{
					Name:       tempoQueryJaegerUiPortName,
					Port:       portJaegerUI,
					TargetPort: intstr.FromInt(portJaegerUI),
				},
				{
					Name:       tempoQueryMetricsPortName,
					Port:       portQueryMetrics,
					TargetPort: intstr.FromString("jaeger-metrics"),
				},
			},
			Selector: labels,
		},
	}

	frontEndDiscoveryService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      manifestutils.Name(componentName+"-discovery", tempo.Name),
			Namespace: tempo.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       httpPortName,
					Port:       portHTTPServer,
					TargetPort: intstr.FromInt(portHTTPServer),
				},
				{
					Name:       grpcPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       portGRPCServer,
					TargetPort: intstr.FromInt(portGRPCServer),
				},
				{
					Name:       tempoQueryJaegerUiPortName,
					Port:       portJaegerUI,
					TargetPort: intstr.FromInt(portJaegerUI),
				},
				{
					Name:       tempoQueryMetricsPortName,
					Port:       portQueryMetrics,
					TargetPort: intstr.FromString("jaeger-metrics"),
				},
				{
					Name:       grpclbPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       portGRPCLBServer,
					TargetPort: intstr.FromString("grpc"),
				},
			},
			Selector: labels,
		},
	}

	services := []*corev1.Service{frontEndService, frontEndDiscoveryService}

	return services
}

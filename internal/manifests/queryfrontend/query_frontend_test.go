package queryfrontend

import (
	"testing"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	configv1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	"github.com/os-observability/tempo-operator/internal/manifests/memberlist"
	"github.com/os-observability/tempo-operator/internal/manifests/naming"
)

func getJaegerServicePorts() []corev1.ServicePort {
	jaegerServicePorts := []corev1.ServicePort{
		{
			Name:       jaegerUIPortName,
			Port:       portJaegerUI,
			TargetPort: intstr.FromString(jaegerUIPortName),
		},
		{
			Name:       jaegerMetricsPortName,
			Port:       portJaegerMetrics,
			TargetPort: intstr.FromString(jaegerMetricsPortName),
		},
	}
	return jaegerServicePorts
}

func getExpectedFrontEndService(withJaeger bool) *corev1.Service {
	labels := manifestutils.ComponentLabels("query-frontend", "test")
	expectedFrontEndService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.QueryFrontendComponentName, "test"),
			Namespace: "project1",
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       manifestutils.HttpPortName,
					Port:       manifestutils.PortHTTPServer,
					TargetPort: intstr.FromString(manifestutils.HttpPortName),
				},
				{
					Name:       manifestutils.GrpcPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortGRPCServer,
					TargetPort: intstr.FromString(manifestutils.GrpcPortName),
				},
			},
			Selector: labels,
		},
	}
	if withJaeger {
		expectedFrontEndService.Spec.Ports = append(expectedFrontEndService.Spec.Ports, getJaegerServicePorts()...)
	}

	return expectedFrontEndService
}

func getExpectedFrontendDiscoveryService(withJaeger bool) *corev1.Service {
	labels := manifestutils.ComponentLabels("query-frontend", "test")

	expectedFrontendDiscoveryService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.QueryFrontendComponentName+"-discovery", "test"),
			Namespace: "project1",
			Labels:    manifestutils.ComponentLabels("query-frontend-discovery", "test"),
		},
		Spec: corev1.ServiceSpec{
			ClusterIP:                "None",
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name:       manifestutils.HttpPortName,
					Port:       manifestutils.PortHTTPServer,
					TargetPort: intstr.FromString(manifestutils.HttpPortName),
				},
				{
					Name:       manifestutils.GrpcPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortGRPCServer,
					TargetPort: intstr.FromString(manifestutils.GrpcPortName),
				},
				{
					Name:       grpclbPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       portGRPCLBServer,
					TargetPort: intstr.FromString(grpclbPortName),
				},
			},
			Selector: labels,
		},
	}
	if withJaeger {
		expectedFrontendDiscoveryService.Spec.Ports = append(expectedFrontendDiscoveryService.Spec.Ports, getJaegerServicePorts()...)
	}

	return expectedFrontendDiscoveryService
}

func getExpectedDeployment(withJaeger bool) *v1.Deployment {
	labels := manifestutils.ComponentLabels("query-frontend", "test")
	annotations := manifestutils.CommonAnnotations("")

	expectedDeployment := &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.QueryFrontendComponentName, "test"),
			Namespace: "project1",
			Labels:    labels,
		},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      k8slabels.Merge(labels, memberlist.GossipSelector),
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "tempo-test-serviceaccount",
					Affinity:           manifestutils.DefaultAffinity(labels),
					Containers: []corev1.Container{
						{
							Name:  "tempo",
							Image: "docker.io/grafana/tempo:1.5.0",
							Args: []string{
								"-target=query-frontend",
								"-config.file=/conf/tempo.yaml",
								"-mem-ballast-size-mbs=1024",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          manifestutils.HttpPortName,
									ContainerPort: manifestutils.PortHTTPServer,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          manifestutils.GrpcPortName,
									ContainerPort: manifestutils.PortGRPCServer,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							ReadinessProbe: manifestutils.TempoReadinessProbe(),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      manifestutils.ConfigVolumeName,
									MountPath: "/conf",
									ReadOnly:  true,
								},
								{
									Name:      manifestutils.TmpStorageVolumeName,
									MountPath: manifestutils.TmpStoragePath,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    *resource.NewMilliQuantity(90, resource.BinarySI),
									corev1.ResourceMemory: *resource.NewQuantity(107374184, resource.BinarySI),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    *resource.NewMilliQuantity(27, resource.BinarySI),
									corev1.ResourceMemory: *resource.NewQuantity(32212256, resource.BinarySI),
								},
							},
							SecurityContext: manifestutils.TempoContainerSecurityContext(),
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: manifestutils.ConfigVolumeName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: naming.Name("", "test"),
									},
								},
							},
						},
						{
							Name: manifestutils.TmpStorageVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	if withJaeger {
		jaegerQueryContainer := corev1.Container{
			Name:  "tempo-query",
			Image: "docker.io/grafana/tempo-query:1.5.0",
			Args: []string{
				"--query.base-path=/",
				"--grpc-storage-plugin.configuration-file=/conf/tempo-query.yaml",
				"--query.bearer-token-propagation=true",
			},
			Ports: []corev1.ContainerPort{
				{
					Name:          jaegerUIPortName,
					ContainerPort: portJaegerUI,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          jaegerMetricsPortName,
					ContainerPort: portJaegerMetrics,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      manifestutils.ConfigVolumeName,
					MountPath: "/conf",
					ReadOnly:  true,
				},
				{
					Name:      manifestutils.TmpStorageVolumeName + "-query",
					MountPath: manifestutils.TmpStoragePath,
				},
			},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(90, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(107374184, resource.BinarySI),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(27, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(32212256, resource.BinarySI),
				},
			},
		}
		jaegerQueryVolume := corev1.Volume{
			Name: manifestutils.TmpStorageVolumeName + "-query",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		//expectedDeployment.Spec.Template.Spec.Containers

		expectedDeployment.Spec.Template.Spec.Containers = append(expectedDeployment.Spec.Template.Spec.Containers, jaegerQueryContainer)
		expectedDeployment.Spec.Template.Spec.Volumes = append(expectedDeployment.Spec.Template.Spec.Volumes, jaegerQueryVolume)
		expectedDeployment.Spec.Template.Spec.NodeSelector = map[string]string{"a": "b"}
		expectedDeployment.Spec.Template.Spec.Tolerations = []corev1.Toleration{{Key: "c"}}
	}

	return expectedDeployment
}

func TestBuildQueryFrontend(t *testing.T) {
	objects, err := BuildQueryFrontend(manifestutils.Params{Tempo: v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "project1",
		},
		Spec: v1alpha1.TempoStackSpec{
			Images: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:1.5.0",
			},
			ServiceAccount: "tempo-test-serviceaccount",
			Resources: v1alpha1.Resources{
				Total: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1000m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				},
			},
		},
	}})
	require.NoError(t, err)
	require.Equal(t, 3, len(objects))

	// Test the services
	frontendService := objects[1].(*corev1.Service)
	expectedFrontEndService := getExpectedFrontEndService(false)
	frontEndDiscoveryService := objects[2].(*corev1.Service)
	expectedFrontendDiscoveryService := getExpectedFrontendDiscoveryService(false)
	assert.Equal(t, expectedFrontendDiscoveryService, frontEndDiscoveryService)
	assert.Equal(t, expectedFrontEndService, frontendService)

	deployment := objects[0].(*v1.Deployment)
	expectedDeployment := getExpectedDeployment(false)
	assert.Equal(t, expectedDeployment, deployment)
}

func TestBuildQueryFrontendWithJaeger(t *testing.T) {
	withJaeger := true
	objects, err := BuildQueryFrontend(manifestutils.Params{Tempo: v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "project1",
		},
		Spec: v1alpha1.TempoStackSpec{
			Images: configv1alpha1.ImagesSpec{
				Tempo:      "docker.io/grafana/tempo:1.5.0",
				TempoQuery: "docker.io/grafana/tempo-query:1.5.0",
			},
			ServiceAccount: "tempo-test-serviceaccount",
			Template: v1alpha1.TempoTemplateSpec{
				QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
					TempoComponentSpec: v1alpha1.TempoComponentSpec{
						NodeSelector: map[string]string{"a": "b"},
						Tolerations: []corev1.Toleration{
							{
								Key: "c",
							},
						},
					},
					JaegerQuery: v1alpha1.JaegerQuerySpec{
						Enabled: true,
					},
				},
			},
			Resources: v1alpha1.Resources{
				Total: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1000m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				},
			},
		},
	}})

	require.NoError(t, err)
	require.Equal(t, 3, len(objects))

	// Test the services
	frontendService := objects[1].(*corev1.Service)

	expectedFrontEndService := getExpectedFrontEndService(withJaeger)
	assert.Equal(t, expectedFrontEndService, frontendService)

	frontEndDiscoveryService := objects[2].(*corev1.Service)
	expectedFrontendDiscoveryService := getExpectedFrontendDiscoveryService(withJaeger)
	assert.Equal(t, expectedFrontendDiscoveryService, frontEndDiscoveryService)

	deployment := objects[0].(*v1.Deployment)
	expectedDeployment := getExpectedDeployment(withJaeger)
	assert.Equal(t, expectedDeployment, deployment)
}

func TestQueryFrontendJaegerIngress(t *testing.T) {
	objects, err := BuildQueryFrontend(manifestutils.Params{Tempo: v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "project1",
		},
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
					JaegerQuery: v1alpha1.JaegerQuerySpec{
						Enabled: true,
						Ingress: v1alpha1.JaegerQueryIngressSpec{
							Type: "ingress",
							Host: "jaeger.example.com",
							Annotations: map[string]string{
								"traefik.ingress.kubernetes.io/router.tls": "true",
							},
						},
					},
				},
			},
		},
	}})

	require.NoError(t, err)
	require.Equal(t, 4, len(objects))
	pathType := networkingv1.PathTypePrefix
	assert.Equal(t, &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.QueryFrontendComponentName, "test"),
			Namespace: "project1",
			Labels:    manifestutils.ComponentLabels("query-frontend", "test"),
			Annotations: map[string]string{
				"traefik.ingress.kubernetes.io/router.tls": "true",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "jaeger.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: naming.Name(manifestutils.QueryFrontendComponentName, "test"),
											Port: networkingv1.ServiceBackendPort{
												Name: jaegerUIPortName,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, objects[3].(*networkingv1.Ingress))
}

func TestQueryFrontendJaegerRoute(t *testing.T) {
	objects, err := BuildQueryFrontend(manifestutils.Params{Tempo: v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "project1",
		},
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
					JaegerQuery: v1alpha1.JaegerQuerySpec{
						Enabled: true,
						Ingress: v1alpha1.JaegerQueryIngressSpec{
							Type: v1alpha1.IngressTypeRoute,
							Route: v1alpha1.JaegerQueryRouteSpec{
								Termination: v1alpha1.TLSRouteTerminationTypeEdge,
							},
						},
					},
				},
			},
		},
	}})

	require.NoError(t, err)
	require.Equal(t, 4, len(objects))
	assert.Equal(t, &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.QueryFrontendComponentName, "test"),
			Namespace: "project1",
			Labels:    manifestutils.ComponentLabels("query-frontend", "test"),
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: naming.Name(manifestutils.QueryFrontendComponentName, "test"),
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString(jaegerUIPortName),
			},
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationEdge,
			},
		},
	}, objects[3].(*routev1.Route))
}

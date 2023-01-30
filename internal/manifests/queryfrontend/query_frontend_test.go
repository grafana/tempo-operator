package queryfrontend

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

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
			Name:      naming.Name(componentName, "test"),
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
			Name:      naming.Name(componentName+"-discovery", "test"),
			Namespace: "project1",
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
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
			Name:      naming.Name(componentName, "test"),
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
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
								{
									Weight: 100,
									PodAffinityTerm: corev1.PodAffinityTerm{
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: labels,
										},
										TopologyKey: "failure-domain.beta.kubernetes.io/zone",
									},
								},
							},
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									LabelSelector: &metav1.LabelSelector{
										MatchLabels: labels,
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "query-frontend",
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
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      manifestutils.ConfigVolumeName,
									MountPath: "/conf",
									ReadOnly:  true,
								},
								{
									Name:      "data-querier-frontend",
									MountPath: "/var/tempo",
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
							Name: "data-querier-frontend",
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
					Name:      "data-query",
					MountPath: "/var/tempo",
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
			Name: "data-query",
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
	objects, err := BuildQueryFrontend(manifestutils.Params{Tempo: v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "project1",
		},
		Spec: v1alpha1.MicroservicesSpec{
			Images: v1alpha1.ImagesSpec{
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
	objects, err := BuildQueryFrontend(manifestutils.Params{Tempo: v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "project1",
		},
		Spec: v1alpha1.MicroservicesSpec{
			Images: v1alpha1.ImagesSpec{
				Tempo:      "docker.io/grafana/tempo:1.5.0",
				TempoQuery: "docker.io/grafana/tempo-query:1.5.0",
			},
			ServiceAccount: "tempo-test-serviceaccount",
			Components: v1alpha1.TempoComponentsSpec{
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

package metricsgenerator

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

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func newParams(tempo v1alpha1.TempoStack) manifestutils.Params {
	return manifestutils.Params{
		Tempo: tempo,
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:latest",
			},
		},
	}
}

func newTempoStack(name, ns string, cfg v1alpha1.TempoMetricsGeneratorSpec) v1alpha1.TempoStack {
	return v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1alpha1.TempoStackSpec{
			ServiceAccount: "tempo-test-serviceaccount",
			Images: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:1.5.0",
			},
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: "test-storage-secret",
					Type: "s3",
				},
			},
			Template: v1alpha1.TempoTemplateSpec{
				MetricsGenerator: &cfg,
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
	}
}

func TestBuildMetricsGenerator(t *testing.T) {
	tempo := newTempoStack("test", "project1", v1alpha1.TempoMetricsGeneratorSpec{
		TempoComponentSpec: v1alpha1.TempoComponentSpec{
			NodeSelector: map[string]string{"a": "b"},
			Tolerations: []corev1.Toleration{
				{Key: "c"},
			},
		},
		RemoteWriteURLs: []string{"http://prometheus:9090/api/v1/write"},
	})

	objects, err := BuildMetricsGenerator(newParams(tempo))
	require.NoError(t, err)
	assert.Len(t, objects, 2)

	labels := manifestutils.ComponentLabels(manifestutils.MetricsGeneratorComponentName, "test")
	annotations := manifestutils.CommonAnnotations("")

	d, ok := objects[0].(*v1.Deployment)
	require.True(t, ok)

	assert.Equal(t, &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-test-metrics-generator",
			Namespace: "project1",
			Labels:    labels,
		},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      k8slabels.Merge(labels, map[string]string{"tempo-gossip-member": "true"}),
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "tempo-test-serviceaccount",
					NodeSelector:       map[string]string{"a": "b"},
					Tolerations: []corev1.Toleration{
						{Key: "c"},
					},
					Affinity: manifestutils.DefaultAffinity(labels),
					Containers: []corev1.Container{
						{
							Name:  "tempo",
							Image: "docker.io/grafana/tempo:1.5.0",
							Args: []string{
								"-target=metrics-generator",
								"-config.file=/conf/tempo.yaml",
								"-log.level=info",
								"-config.expand-env=true",
							},
							Env: d.Spec.Template.Spec.Containers[0].Env,
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
								{
									Name:          manifestutils.HttpMemberlistPortName,
									ContainerPort: manifestutils.PortMemberlist,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Scheme: corev1.URISchemeHTTP,
										Path:   manifestutils.TempoReadinessPath,
										Port:   intstr.FromString(manifestutils.HttpPortName),
									},
								},
								InitialDelaySeconds: 15,
								TimeoutSeconds:      1,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      manifestutils.ConfigVolumeName,
									MountPath: "/conf",
									ReadOnly:  true,
								},
								{
									Name:      manifestutils.TmpStorageVolumeName,
									MountPath: manifestutils.TmpTempoStoragePath,
								},
							},
							Resources:       resources(tempo),
							SecurityContext: manifestutils.TempoContainerSecurityContext(),
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: manifestutils.ConfigVolumeName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "tempo-test",
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
	}, d)

	svc, ok := objects[1].(*corev1.Service)
	require.True(t, ok)
	assert.Equal(t, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-test-metrics-generator",
			Namespace: "project1",
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       manifestutils.HttpMemberlistPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortMemberlist,
					TargetPort: intstr.FromString(manifestutils.HttpMemberlistPortName),
				},
				{
					Name:       manifestutils.HttpPortName,
					Protocol:   corev1.ProtocolTCP,
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
	}, svc)
}

func TestBuildMetricsGeneratorUsesDefaultImage(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "project1"},
		Spec: v1alpha1.TempoStackSpec{
			ServiceAccount: "tempo-test-serviceaccount",
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{Name: "test-storage-secret", Type: "s3"},
			},
			Template: v1alpha1.TempoTemplateSpec{
				MetricsGenerator: &v1alpha1.TempoMetricsGeneratorSpec{
					RemoteWriteURLs: []string{"http://prometheus:9090/api/v1/write"},
				},
			},
		},
	}

	params := manifestutils.Params{
		Tempo: tempo,
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:default",
			},
		},
	}

	objects, err := BuildMetricsGenerator(params)
	require.NoError(t, err)
	require.Len(t, objects, 2)

	d, ok := objects[0].(*v1.Deployment)
	require.True(t, ok)
	assert.Equal(t, "docker.io/grafana/tempo:default", d.Spec.Template.Spec.Containers[0].Image)
}

func TestBuildMetricsGeneratorOverrideResources(t *testing.T) {
	overrideResources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}

	tempo := newTempoStack("test", "project1", v1alpha1.TempoMetricsGeneratorSpec{
		TempoComponentSpec: v1alpha1.TempoComponentSpec{
			Resources: &overrideResources,
		},
		RemoteWriteURLs: []string{"http://prometheus:9090/api/v1/write"},
	})

	objects, err := BuildMetricsGenerator(newParams(tempo))
	require.NoError(t, err)
	require.Len(t, objects, 2)

	d, ok := objects[0].(*v1.Deployment)
	require.True(t, ok)
	assert.Equal(t, overrideResources, d.Spec.Template.Spec.Containers[0].Resources)
}

func TestBuildMetricsGeneratorHashRingPodIP(t *testing.T) {
	tempo := newTempoStack("test", "project1", v1alpha1.TempoMetricsGeneratorSpec{
		RemoteWriteURLs: []string{"http://prometheus:9090/api/v1/write"},
	})
	tempo.Spec.HashRing = v1alpha1.HashRingSpec{
		MemberList: v1alpha1.MemberListSpec{
			InstanceAddrType: v1alpha1.InstanceAddrPodIP,
		},
	}

	objects, err := BuildMetricsGenerator(newParams(tempo))
	require.NoError(t, err)
	require.Len(t, objects, 2)

	d, ok := objects[0].(*v1.Deployment)
	require.True(t, ok)

	envVars := d.Spec.Template.Spec.Containers[0].Env
	var hashRingEnv *corev1.EnvVar
	for i := range envVars {
		if envVars[i].Name == "HASH_RING_INSTANCE_ADDR" {
			hashRingEnv = &envVars[i]
			break
		}
	}
	require.NotNil(t, hashRingEnv)
	assert.Equal(t, &corev1.ObjectFieldSelector{
		APIVersion: "v1",
		FieldPath:  "status.podIP",
	}, hashRingEnv.ValueFrom.FieldRef)
}

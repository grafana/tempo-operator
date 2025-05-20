package compactor

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
	"k8s.io/utils/ptr"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildCompactor(t *testing.T) {

	tests := []struct {
		name                     string
		instanceAddrType         v1alpha1.InstanceAddrType
		expectedContainerEnvVars []corev1.EnvVar
	}{
		{
			name: "default",
			expectedContainerEnvVars: []corev1.EnvVar{
				{
					Name:  "GOMEMLIMIT",
					Value: "154618828",
				},
			},
		},
		{
			name: "set InstanceAddrType to PodIP",
			expectedContainerEnvVars: []corev1.EnvVar{
				{
					Name:  "GOMEMLIMIT",
					Value: "154618828",
				},
				{
					Name: "HASH_RING_INSTANCE_ADDR",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "status.podIP",
						},
					},
				},
			},
			instanceAddrType: v1alpha1.InstanceAddrPodIP,
		},
	}

	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {

			instanceAddrType := v1alpha1.InstanceAddrDefault
			if ts.instanceAddrType != "" {
				instanceAddrType = ts.instanceAddrType
			}

			objects, err := BuildCompactor(manifestutils.Params{Tempo: v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "project1",
				},
				Spec: v1alpha1.TempoStackSpec{
					Images: configv1alpha1.ImagesSpec{
						Tempo: "docker.io/grafana/tempo:1.5.0",
					},
					ServiceAccount: "tempo-test-serviceaccount",
					Template: v1alpha1.TempoTemplateSpec{
						Compactor: v1alpha1.TempoComponentSpec{
							Replicas:     ptr.To(int32(2)),
							NodeSelector: map[string]string{"a": "b"},
							Tolerations: []corev1.Toleration{
								{
									Key: "c",
								},
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
					HashRing: v1alpha1.HashRingSpec{
						MemberList: v1alpha1.MemberListSpec{
							InstanceAddrType: instanceAddrType,
						},
					},
				},
			}})
			require.NoError(t, err)

			labels := manifestutils.ComponentLabels("compactor", "test")
			annotations := manifestutils.CommonAnnotations("")
			assert.Equal(t, 2, len(objects))

			assert.Equal(t, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tempo-test-compactor",
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
					},
					Selector: labels,
				},
			}, objects[1])

			assert.Equal(t, &v1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tempo-test-compactor",
					Namespace: "project1",
					Labels:    labels,
				},
				Spec: v1.DeploymentSpec{
					Replicas: ptr.To(int32(2)),
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
								{
									Key: "c",
								},
							},
							Containers: []corev1.Container{
								{
									Name:  "tempo",
									Image: "docker.io/grafana/tempo:1.5.0",
									Env:   ts.expectedContainerEnvVars,
									Args: []string{
										"-target=compactor",
										"-config.file=/conf/tempo.yaml",
										"-log.level=info",
										"-config.expand-env=true",
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
									Ports: []corev1.ContainerPort{
										{
											Name:          manifestutils.HttpPortName,
											ContainerPort: manifestutils.PortHTTPServer,
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
									Resources: corev1.ResourceRequirements{
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    *resource.NewMilliQuantity(80, resource.BinarySI),
											corev1.ResourceMemory: *resource.NewQuantity(193273536, resource.BinarySI),
										},
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    *resource.NewMilliQuantity(24, resource.BinarySI),
											corev1.ResourceMemory: *resource.NewQuantity(57982064, resource.BinarySI),
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
			}, objects[0])
		})
	}
}

func TestOverrideResources(t *testing.T) {
	overrideResources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
	}

	objects, err := BuildCompactor(manifestutils.Params{Tempo: v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "project1",
		},
		Spec: v1alpha1.TempoStackSpec{
			Images: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:1.5.0",
			},
			ServiceAccount: "tempo-test-serviceaccount",
			Template: v1alpha1.TempoTemplateSpec{
				Compactor: v1alpha1.TempoComponentSpec{
					Replicas:     ptr.To(int32(2)),
					NodeSelector: map[string]string{"a": "b"},
					Tolerations: []corev1.Toleration{
						{
							Key: "c",
						},
					},
					Resources: &overrideResources,
				},
			},
			Resources: v1alpha1.Resources{
				Total: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("10000m"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
			},
		},
	}})
	require.NoError(t, err)
	dep, ok := objects[0].(*v1.Deployment)
	require.True(t, ok)
	assert.Equal(t, dep.Spec.Template.Spec.Containers[0].Resources, overrideResources)
}

package ingester

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

func TestBuildIngester(t *testing.T) {

	tests := []struct {
		name                     string
		instanceAddrType         v1alpha1.InstanceAddrType
		expectedContainerEnvVars []corev1.EnvVar
	}{
		{
			name: "default",
			expectedContainerEnvVars: []corev1.EnvVar{
				{
					Name: "S3_SECRET_KEY",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							Key: "access_key_secret",
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-storage-secret",
							},
						},
					},
				}, {
					Name: "S3_ACCESS_KEY",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							Key: "access_key_id",
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-storage-secret",
							},
						},
					},
				},
				{
					Name:  "GOMEMLIMIT",
					Value: "858993459",
				},
			},
		},
		{
			name:             "set InstanceAddrType to PodIP",
			instanceAddrType: v1alpha1.InstanceAddrPodIP,
			expectedContainerEnvVars: []corev1.EnvVar{
				{
					Name: "S3_SECRET_KEY",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							Key: "access_key_secret",
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-storage-secret",
							},
						},
					},
				}, {
					Name: "S3_ACCESS_KEY",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							Key: "access_key_id",
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-storage-secret",
							},
						},
					},
				},
				{
					Name:  "GOMEMLIMIT",
					Value: "858993459",
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
		},
	}
	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {
			storageClassName := "default"
			filesystem := corev1.PersistentVolumeFilesystem

			instanceAddrType := v1alpha1.InstanceAddrDefault
			if ts.instanceAddrType != "" {
				instanceAddrType = ts.instanceAddrType
			}

			objects, err := BuildIngester(manifestutils.Params{Tempo: v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "project1",
				},
				Spec: v1alpha1.TempoStackSpec{
					Images: configv1alpha1.ImagesSpec{
						Tempo: "docker.io/grafana/tempo:1.5.0",
					},
					ServiceAccount: "tempo-test-serviceaccount",
					Storage: v1alpha1.ObjectStorageSpec{
						Secret: v1alpha1.ObjectStorageSecretSpec{
							CredentialMode: v1alpha1.CredentialModeStatic,
							Name:           "test-storage-secret",
							Type:           "s3",
						},
					},
					StorageSize:      resource.MustParse("10Gi"),
					StorageClassName: &storageClassName,
					Template: v1alpha1.TempoTemplateSpec{
						Ingester: v1alpha1.TempoComponentSpec{
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

			labels := manifestutils.ComponentLabels("ingester", "test")
			annotations := manifestutils.CommonAnnotations("")
			assert.Equal(t, 2, len(objects))

			assert.Equal(t, &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tempo-test-ingester",
					Namespace: "project1",
					Labels:    labels,
				},
				Spec: v1.StatefulSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: labels,
					},
					PodManagementPolicy: v1.ParallelPodManagement,
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
							Affinity: manifestutils.DefaultAffinity(labels),
							Containers: []corev1.Container{
								{
									Name:  "tempo",
									Image: "docker.io/grafana/tempo:1.5.0",
									Args: []string{"-target=ingester",
										"-config.file=/conf/tempo.yaml",
										"-log.level=info",
										"-config.expand-env=true",
										"--storage.trace.s3.secret_key=$(S3_SECRET_KEY)",
										"--storage.trace.s3.access_key=$(S3_ACCESS_KEY)",
									},
									Env: ts.expectedContainerEnvVars,
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      manifestutils.ConfigVolumeName,
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
											Name:          manifestutils.HttpMemberlistPortName,
											ContainerPort: manifestutils.PortMemberlist,
											Protocol:      corev1.ProtocolTCP,
										},
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
											corev1.ResourceCPU:    *resource.NewMilliQuantity(380, resource.BinarySI),
											corev1.ResourceMemory: *resource.NewQuantity(1073741824, resource.BinarySI),
										},
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    *resource.NewMilliQuantity(114, resource.BinarySI),
											corev1.ResourceMemory: *resource.NewQuantity(322122560, resource.BinarySI),
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
								Resources: corev1.VolumeResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("10Gi"),
									},
								},
								StorageClassName: &storageClassName,
								VolumeMode:       &filesystem,
							},
						},
					},
				},
			}, objects[0])
			assert.Equal(t, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tempo-test-ingester",
					Namespace: "project1",
					Labels:    labels,
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
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
			}, objects[1])
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

	objects, err := BuildIngester(manifestutils.Params{Tempo: v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "project1",
		},
		Spec: v1alpha1.TempoStackSpec{
			Images: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:1.5.0",
			},
			ServiceAccount: "tempo-test-serviceaccount",
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					CredentialMode: v1alpha1.CredentialModeStatic,
					Name:           "test-storage-secret",
					Type:           "s3",
				},
			},
			StorageSize: resource.MustParse("10Gi"),
			Template: v1alpha1.TempoTemplateSpec{
				Ingester: v1alpha1.TempoComponentSpec{
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
						corev1.ResourceCPU:    resource.MustParse("1000m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				},
			},
		},
	}})
	require.NoError(t, err)
	ss, ok := objects[0].(*v1.StatefulSet)
	require.True(t, ok)
	assert.Equal(t, ss.Spec.Template.Spec.Containers[0].Resources, overrideResources)
}

package monolithic

import (
	"testing"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/operator-framework/operator-lib/proxy"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var (
	oneGBQuantity = resource.MustParse("1Gi")
	tenGBQuantity = resource.MustParse("10Gi")
)

func TestStatefulsetMemoryStorage(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:x.y.z",
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Storage: v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: "memory",
					},
				},
				Ingestion: &v1alpha1.MonolithicIngestionSpec{
					OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
						GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
							Enabled: true,
						},
					},
				},
			},
		},
	}
	sts, err := BuildTempoStatefulset(opts)
	require.NoError(t, err)

	labels := ComponentLabels("tempo", "sample")
	annotations := manifestutils.CommonAnnotations("")
	require.Equal(t, &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-sample",
			Namespace: "default",
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					Affinity: manifestutils.DefaultAffinity(labels),
					Containers: []corev1.Container{
						{
							Name:  "tempo",
							Image: "docker.io/grafana/tempo:x.y.z",
							Env:   proxy.ReadProxyVarsFromEnv(),
							Args: []string{
								"-config.file=/conf/tempo.yaml",
								"-mem-ballast-size-mbs=1024",
								"-log.level=info",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "tempo-conf",
									MountPath: "/conf",
									ReadOnly:  true,
								},
								{
									Name:      "tempo-wal",
									MountPath: "/var/tempo/wal",
								},
								{
									Name:      "tempo-blocks",
									MountPath: "/var/tempo/blocks",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 3200,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "otlp-grpc",
									ContainerPort: 4317,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							ReadinessProbe:  manifestutils.TempoReadinessProbe(false),
							SecurityContext: manifestutils.TempoContainerSecurityContext(),
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "tempo-conf",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "tempo-sample",
									},
								},
							},
						},
						{
							Name: "tempo-wal",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									Medium: corev1.StorageMediumMemory,
								},
							},
						},
						{
							Name: "tempo-blocks",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									Medium: corev1.StorageMediumMemory,
								},
							},
						},
					},
				},
			},
		},
	}, sts)
}

func TestStatefulsetPVStorage(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:x.y.z",
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Storage: v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: "pv",
						WAL: &v1alpha1.MonolithicTracesStorageWALSpec{
							Size: oneGBQuantity,
						},
						PV: &v1alpha1.MonolithicTracesStoragePVSpec{
							Size: tenGBQuantity,
						},
					},
				},
			},
		},
	}
	sts, err := BuildTempoStatefulset(opts)
	require.NoError(t, err)

	require.Equal(t, []corev1.VolumeMount{
		{
			Name:      "tempo-conf",
			MountPath: "/conf",
			ReadOnly:  true,
		},
		{
			Name:      "tempo-wal",
			MountPath: "/var/tempo/wal",
		},
		{
			Name:      "tempo-blocks",
			MountPath: "/var/tempo/blocks",
		},
	}, sts.Spec.Template.Spec.Containers[0].VolumeMounts)

	require.Equal(t, []corev1.Volume{
		{
			Name: "tempo-conf",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "tempo-sample",
					},
				},
			},
		},
	}, sts.Spec.Template.Spec.Volumes)

	require.Equal(t, []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "tempo-wal",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: oneGBQuantity,
					},
				},
				VolumeMode: ptr.To(corev1.PersistentVolumeFilesystem),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "tempo-blocks",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: tenGBQuantity,
					},
				},
				VolumeMode: ptr.To(corev1.PersistentVolumeFilesystem),
			},
		},
	}, sts.Spec.VolumeClaimTemplates)
}

func TestStatefulsetPorts(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:x.y.z",
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Storage: v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: "memory",
					},
				},
			},
		},
	}

	tests := []struct {
		name     string
		input    *v1alpha1.MonolithicIngestionSpec
		expected []corev1.ContainerPort
	}{
		{
			name:  "no ingestion ports",
			input: nil,
			expected: []corev1.ContainerPort{
				{
					Name:          "http",
					ContainerPort: 3200,
					Protocol:      corev1.ProtocolTCP,
				},
			},
		},
		{
			name: "OTLP/gRPC",
			input: &v1alpha1.MonolithicIngestionSpec{
				OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
					GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
						Enabled: true,
					},
				},
			},
			expected: []corev1.ContainerPort{
				{
					Name:          "http",
					ContainerPort: 3200,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "otlp-grpc",
					ContainerPort: 4317,
					Protocol:      corev1.ProtocolTCP,
				},
			},
		},
		{
			name: "OTLP/HTTP",
			input: &v1alpha1.MonolithicIngestionSpec{
				OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
					HTTP: &v1alpha1.MonolithicIngestionOTLPProtocolsHTTPSpec{
						Enabled: true,
					},
				},
			},
			expected: []corev1.ContainerPort{
				{
					Name:          "http",
					ContainerPort: 3200,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "otlp-http",
					ContainerPort: 4318,
					Protocol:      corev1.ProtocolTCP,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts.Tempo.Spec.Ingestion = test.input
			sts, err := BuildTempoStatefulset(opts)
			require.NoError(t, err)
			require.Equal(t, test.expected, sts.Spec.Template.Spec.Containers[0].Ports)
		})
	}
}

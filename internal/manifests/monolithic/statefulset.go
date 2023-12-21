package monolithic

import (
	"errors"
	"fmt"

	"github.com/operator-framework/operator-lib/proxy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

const (
	walVolumeName    = "tempo-wal"
	blocksVolumeName = "tempo-blocks"
)

// BuildTempoStatefulset creates the Tempo statefulset for a monolithic deployment.
func BuildTempoStatefulset(opts Options) (*appsv1.StatefulSet, error) {
	tempo := opts.Tempo
	labels := ComponentLabels("tempo", tempo.Name)
	annotations := manifestutils.CommonAnnotations(opts.ConfigChecksum)

	ss := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name("", tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},

			// Changes to a StatefulSet are not propagated to pods in a broken state (e.g. CrashLoopBackOff)
			// See https://github.com/kubernetes/kubernetes/issues/67250
			//
			// This is a workaround for the above issue.
			// This setting is also in the tempo-distributed helm chart: https://github.com/grafana/helm-charts/blob/0fdf2e1900733eb104ac734f5fb0a89dc950d2c2/charts/tempo-distributed/templates/ingester/statefulset-ingester.yaml#L21
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
							Image: opts.CtrlConfig.DefaultImages.Tempo,
							Env:   proxy.ReadProxyVarsFromEnv(),
							Args: []string{
								"-config.file=/conf/tempo.yaml",
								"-mem-ballast-size-mbs=1024",
								"-log.level=info",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      manifestutils.ConfigVolumeName,
									MountPath: "/conf",
									ReadOnly:  true,
								},
								{
									Name:      walVolumeName,
									MountPath: "/var/tempo/wal",
								},
							},
							Ports:           buildTempoPorts(opts),
							ReadinessProbe:  manifestutils.TempoReadinessProbe(false),
							SecurityContext: manifestutils.TempoContainerSecurityContext(),
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: manifestutils.ConfigVolumeName,
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
		},
	}

	err := configureStorage(opts, ss)
	if err != nil {
		return nil, err
	}

	if tempo.Spec.JaegerUI != nil && tempo.Spec.JaegerUI.Enabled {
		configureJaegerUI(opts, ss)
	}

	return ss, nil
}

func buildTempoPorts(opts Options) []corev1.ContainerPort {
	tempo := opts.Tempo
	ports := []corev1.ContainerPort{
		{
			Name:          manifestutils.HttpPortName,
			ContainerPort: manifestutils.PortHTTPServer,
			Protocol:      corev1.ProtocolTCP,
		},
	}

	if tempo.Spec.Ingestion != nil && tempo.Spec.Ingestion.OTLP != nil {
		if tempo.Spec.Ingestion.OTLP.GRPC != nil && tempo.Spec.Ingestion.OTLP.GRPC.Enabled {
			ports = append(ports, corev1.ContainerPort{
				Name:          manifestutils.OtlpGrpcPortName,
				ContainerPort: manifestutils.PortOtlpGrpcServer,
				Protocol:      corev1.ProtocolTCP,
			})
		}
		if tempo.Spec.Ingestion.OTLP.HTTP != nil && tempo.Spec.Ingestion.OTLP.HTTP.Enabled {
			ports = append(ports, corev1.ContainerPort{
				Name:          manifestutils.PortOtlpHttpName,
				ContainerPort: manifestutils.PortOtlpHttp,
				Protocol:      corev1.ProtocolTCP,
			})
		}
	}

	return ports
}

func configureStorage(opts Options, sts *appsv1.StatefulSet) error {
	tempo := opts.Tempo
	switch tempo.Spec.Storage.Traces.Backend {
	case v1alpha1.MonolithicTracesStorageBackendMemory:
		sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: walVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		})

		sts.Spec.Template.Spec.Containers[0].VolumeMounts = append(sts.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      blocksVolumeName,
			MountPath: "/var/tempo/blocks",
		})
		sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: blocksVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		})

	case v1alpha1.MonolithicTracesStorageBackendPersistentVolume:
		if tempo.Spec.Storage.Traces.WAL == nil {
			return errors.New("please configure .spec.storage.traces.wal")
		}
		sts.Spec.VolumeClaimTemplates = append(sts.Spec.VolumeClaimTemplates, corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: walVolumeName,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: tempo.Spec.Storage.Traces.WAL.Size,
					},
				},
				VolumeMode: ptr.To(corev1.PersistentVolumeFilesystem),
			},
		})

		if tempo.Spec.Storage.Traces.PV == nil {
			return errors.New("please configure .spec.storage.traces.pv")
		}
		sts.Spec.Template.Spec.Containers[0].VolumeMounts = append(sts.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      blocksVolumeName,
			MountPath: "/var/tempo/blocks",
		})
		sts.Spec.VolumeClaimTemplates = append(sts.Spec.VolumeClaimTemplates, corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: blocksVolumeName,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: tempo.Spec.Storage.Traces.PV.Size,
					},
				},
				VolumeMode: ptr.To(corev1.PersistentVolumeFilesystem),
			},
		})

	default:
		return fmt.Errorf("invalid storage backend: '%s'", tempo.Spec.Storage.Traces.Backend)
	}
	return nil
}

func configureJaegerUI(opts Options, sts *appsv1.StatefulSet) {
	tempoQuery := corev1.Container{
		Name:  "tempo-query",
		Image: opts.CtrlConfig.DefaultImages.TempoQuery,
		Env:   proxy.ReadProxyVarsFromEnv(),
		Args: []string{
			"--query.base-path=/",
			"--grpc-storage-plugin.configuration-file=/conf/tempo-query.yaml",
			"--query.bearer-token-propagation=true",
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          manifestutils.JaegerGRPCQuery,
				ContainerPort: manifestutils.PortJaegerGRPCQuery,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          manifestutils.JaegerUIPortName,
				ContainerPort: manifestutils.PortJaegerUI,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          manifestutils.JaegerMetricsPortName,
				ContainerPort: manifestutils.PortJaegerMetrics,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      manifestutils.ConfigVolumeName,
				MountPath: "/conf",
				ReadOnly:  true,
			},
		},
	}

	sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, tempoQuery)
}

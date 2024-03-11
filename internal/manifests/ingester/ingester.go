package ingester

import (
	"github.com/operator-framework/operator-lib/proxy"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/memberlist"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

const (
	dataVolumeName = "data"
)

// BuildIngester creates distributor objects.
func BuildIngester(params manifestutils.Params) ([]client.Object, error) {
	ss, err := statefulSet(params)

	if err != nil {
		return nil, err
	}

	gates := params.CtrlConfig.Gates
	tempo := params.Tempo

	if gates.HTTPEncryption || gates.GRPCEncryption {
		caBundleName := naming.SigningCABundleName(tempo.Name)
		if err := manifestutils.ConfigureServiceCA(&ss.Spec.Template.Spec, caBundleName); err != nil {
			return nil, err
		}

		err := manifestutils.ConfigureServicePKI(tempo.Name, manifestutils.IngesterComponentName, &ss.Spec.Template.Spec)
		if err != nil {
			return nil, err
		}
	}

	return []client.Object{ss, service(tempo)}, nil
}

func statefulSet(params manifestutils.Params) (*v1.StatefulSet, error) {
	tempo := params.Tempo
	labels := manifestutils.ComponentLabels(manifestutils.IngesterComponentName, tempo.Name)
	annotations := manifestutils.CommonAnnotations(params.ConfigChecksum)
	filesystem := corev1.PersistentVolumeFilesystem
	cfg := tempo.Spec.Template.Ingester
	image := tempo.Spec.Images.Tempo
	if image == "" {
		image = params.CtrlConfig.DefaultImages.Tempo
	}

	ss := &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.IngesterComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: v1.StatefulSetSpec{
			Replicas: tempo.Spec.Template.Ingester.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},

			// Changes to a StatefulSet are not propagated to pods in a broken state (e.g. CrashLoopBackOff)
			// See https://github.com/kubernetes/kubernetes/issues/67250
			//
			// This is a workaround for the above issue.
			// This setting is also in the tempo-distributed helm chart: https://github.com/grafana/helm-charts/blob/0fdf2e1900733eb104ac734f5fb0a89dc950d2c2/charts/tempo-distributed/templates/ingester/statefulset-ingester.yaml#L21
			PodManagementPolicy: v1.ParallelPodManagement,

			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      k8slabels.Merge(labels, memberlist.GossipSelector),
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: tempo.Spec.ServiceAccount,
					NodeSelector:       cfg.NodeSelector,
					Tolerations:        cfg.Tolerations,
					Affinity:           manifestutils.DefaultAffinity(labels),
					Containers: []corev1.Container{
						{
							Name:  "tempo",
							Image: image,
							Env:   proxy.ReadProxyVarsFromEnv(),
							Args: []string{
								"-target=ingester",
								"-config.file=/conf/tempo.yaml",
								"-log.level=info",
							},
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
							ReadinessProbe:  manifestutils.TempoReadinessProbe(params.CtrlConfig.Gates.HTTPEncryption),
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

	err := manifestutils.ConfigureStorage(tempo, &ss.Spec.Template.Spec, "tempo")
	if err != nil {
		return nil, err
	}

	ss.Spec.Template, err = manifestutils.PatchTracingJaegerEnv(tempo, ss.Spec.Template)
	if err != nil {
		return nil, err
	}

	return ss, nil
}

func resources(tempo v1alpha1.TempoStack) corev1.ResourceRequirements {
	if tempo.Spec.Template.Ingester.Resources == nil {
		return manifestutils.Resources(tempo, manifestutils.IngesterComponentName, tempo.Spec.Template.Ingester.Replicas)
	}
	return *tempo.Spec.Template.Ingester.Resources
}

func service(tempo v1alpha1.TempoStack) *corev1.Service {
	labels := manifestutils.ComponentLabels(manifestutils.IngesterComponentName, tempo.Name)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.IngesterComponentName, tempo.Name),
			Namespace: tempo.Namespace,
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
	}
}

package ingester

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
	"github.com/os-observability/tempo-operator/internal/manifests/naming"
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

	gates := params.Gates
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
							Image: tempo.Spec.Images.Tempo,
							Args:  []string{"-target=ingester", "-config.file=/conf/tempo.yaml"},
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
							ReadinessProbe:  manifestutils.TempoReadinessProbe(params.Gates.HTTPEncryption || params.Gates.GRPCEncryption),
							Resources:       manifestutils.Resources(tempo, manifestutils.IngesterComponentName),
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

	err := manifestutils.ConfigureStorage(tempo, &ss.Spec.Template.Spec)
	if err != nil {
		return nil, err
	}

	ss.Spec.Template, err = manifestutils.PatchTracingJaegerEnv(tempo, ss.Spec.Template)
	if err != nil {
		return nil, err
	}

	return ss, nil
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

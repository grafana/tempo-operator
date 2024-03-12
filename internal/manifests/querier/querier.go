package querier

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

// BuildQuerier creates querier objects.
func BuildQuerier(params manifestutils.Params) ([]client.Object, error) {
	d, err := deployment(params)
	if err != nil {
		return nil, err
	}

	d.Spec.Template, err = manifestutils.PatchTracingJaegerEnv(params.Tempo, d.Spec.Template)
	if err != nil {
		return nil, err
	}

	gates := params.CtrlConfig.Gates
	tempo := params.Tempo

	if gates.HTTPEncryption || gates.GRPCEncryption {
		caBundleName := naming.SigningCABundleName(tempo.Name)
		if err := manifestutils.ConfigureServiceCA(&d.Spec.Template.Spec, caBundleName); err != nil {
			return nil, err
		}
		err := manifestutils.ConfigureServicePKI(tempo.Name, manifestutils.QuerierComponentName, &d.Spec.Template.Spec)
		if err != nil {
			return nil, err
		}
	}

	return []client.Object{d, service(tempo)}, nil
}

func resources(tempo v1alpha1.TempoStack) corev1.ResourceRequirements {
	if tempo.Spec.Template.Querier.Resources == nil {
		return manifestutils.Resources(tempo, manifestutils.QuerierComponentName, tempo.Spec.Template.Querier.Replicas)
	}
	return *tempo.Spec.Template.Querier.Resources
}

func deployment(params manifestutils.Params) (*v1.Deployment, error) {
	tempo := params.Tempo
	labels := manifestutils.ComponentLabels(manifestutils.QuerierComponentName, tempo.Name)
	annotations := manifestutils.CommonAnnotations(params.ConfigChecksum)
	cfg := tempo.Spec.Template.Querier
	image := tempo.Spec.Images.Tempo
	if image == "" {
		image = params.CtrlConfig.DefaultImages.Tempo
	}

	d := &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.QuerierComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: v1.DeploymentSpec{
			Replicas: tempo.Spec.Template.Querier.Replicas,
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
							Image: image,
							Env:   proxy.ReadProxyVarsFromEnv(),
							Args: []string{
								"-target=querier",
								"-config.file=/conf/tempo.yaml",
								"-log.level=info",
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
							ReadinessProbe: manifestutils.TempoReadinessProbe(params.CtrlConfig.Gates.HTTPEncryption),
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

	err := manifestutils.ConfigureStorage(tempo, &d.Spec.Template.Spec, "tempo")
	if err != nil {
		return nil, err
	}
	return d, nil
}

func service(tempo v1alpha1.TempoStack) *corev1.Service {
	labels := manifestutils.ComponentLabels(manifestutils.QuerierComponentName, tempo.Name)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.QuerierComponentName, tempo.Name),
			Namespace: tempo.Namespace,
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
	}
}

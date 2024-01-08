package distributor

import (
	"github.com/ViaQ/logerr/v2/kverrors"
	"github.com/imdario/mergo"
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

// BuildDistributor creates distributor objects.
func BuildDistributor(params manifestutils.Params) ([]client.Object, error) {
	dep := deployment(params)
	var err error
	dep.Spec.Template, err = manifestutils.PatchTracingJaegerEnv(params.Tempo, dep.Spec.Template)
	if err != nil {
		return nil, err
	}
	gates := params.CtrlConfig.Gates
	tempo := params.Tempo
	if gates.HTTPEncryption || gates.GRPCEncryption {
		caBundleName := naming.SigningCABundleName(tempo.Name)
		if err := manifestutils.ConfigureServiceCA(&dep.Spec.Template.Spec, caBundleName); err != nil {
			return nil, err
		}
		err := manifestutils.ConfigureServicePKI(tempo.Name, manifestutils.DistributorComponentName, &dep.Spec.Template.Spec)
		if err != nil {
			return nil, err
		}
	}

	if tempo.Spec.Template.Distributor.TLS.Enabled {
		err = configureReceiversTLS(dep, tempo)
		if err != nil {
			return nil, err
		}
	}

	return []client.Object{dep, service(tempo)}, nil
}

func configureReceiversTLS(dep *v1.Deployment, tempo v1alpha1.TempoStack) error {
	caSecretName := tempo.Spec.Template.Distributor.TLS.CA
	certSecretName := tempo.Spec.Template.Distributor.TLS.Cert
	podSpec := &dep.Spec.Template.Spec
	if caSecretName != "" {
		/*Configure CA*/
		secretCAVolumeSpec := corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: caSecretName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: caSecretName,
							},
						},
					},
				},
			},
		}

		secretCAContainerSpec := corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      caSecretName,
					ReadOnly:  true,
					MountPath: manifestutils.CAReceiver,
				},
			},
		}
		if err := mergo.Merge(podSpec, secretCAVolumeSpec, mergo.WithAppendSlice); err != nil {
			return kverrors.Wrap(err, "failed to merge volumes")
		}

		if err := mergo.Merge(&podSpec.Containers[0], secretCAContainerSpec, mergo.WithAppendSlice); err != nil {
			return kverrors.Wrap(err, "failed to merge container")
		}
	}

	secretCertVolumeSpec := corev1.PodSpec{
		Volumes: []corev1.Volume{
			{
				Name: certSecretName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: certSecretName,
					},
				},
			},
		},
	}
	secretCertContainerSpec := corev1.Container{
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      certSecretName,
				ReadOnly:  true,
				MountPath: manifestutils.TempoReceiverTLSDir(),
			},
		},
	}

	/*Configure certificate*/

	if err := mergo.Merge(podSpec, secretCertVolumeSpec, mergo.WithAppendSlice); err != nil {
		return kverrors.Wrap(err, "failed to merge volumes")
	}

	if err := mergo.Merge(&podSpec.Containers[0], secretCertContainerSpec, mergo.WithAppendSlice); err != nil {
		return kverrors.Wrap(err, "failed to merge container")
	}
	return nil
}

func deployment(params manifestutils.Params) *v1.Deployment {
	tempo := params.Tempo
	labels := manifestutils.ComponentLabels(manifestutils.DistributorComponentName, tempo.Name)
	annotations := manifestutils.CommonAnnotations(params.ConfigChecksum)
	cfg := tempo.Spec.Template.Distributor
	image := tempo.Spec.Images.Tempo
	if image == "" {
		image = params.CtrlConfig.DefaultImages.Tempo
	}

	containerPorts := []corev1.ContainerPort{
		{
			Name:          manifestutils.OtlpGrpcPortName,
			ContainerPort: manifestutils.PortOtlpGrpcServer,
			Protocol:      corev1.ProtocolTCP,
		},
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
	}

	if !tempo.Spec.Template.Gateway.Enabled {
		containerPorts = append(containerPorts, []corev1.ContainerPort{
			{
				Name:          manifestutils.PortOtlpHttpName,
				ContainerPort: manifestutils.PortOtlpHttp,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          manifestutils.PortJaegerThriftHTTPName,
				ContainerPort: manifestutils.PortJaegerThriftHTTP,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          manifestutils.PortJaegerThriftCompactName,
				ContainerPort: manifestutils.PortJaegerThriftCompact,
				Protocol:      corev1.ProtocolUDP,
			},
			{
				Name:          manifestutils.PortJaegerThriftBinaryName,
				ContainerPort: manifestutils.PortJaegerThriftBinary,
				Protocol:      corev1.ProtocolUDP,
			},
			{
				Name:          manifestutils.PortJaegerGrpcName,
				ContainerPort: manifestutils.PortJaegerGrpc,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          manifestutils.PortZipkinName,
				ContainerPort: manifestutils.PortZipkin,
				Protocol:      corev1.ProtocolTCP,
			},
		}...)
	}

	return &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.DistributorComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: v1.DeploymentSpec{
			Replicas: tempo.Spec.Template.Distributor.Replicas,
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
								"-target=distributor",
								"-config.file=/conf/tempo.yaml",
								"-log.level=info",
							},
							Ports:          containerPorts,
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
							Resources:       manifestutils.Resources(tempo, manifestutils.DistributorComponentName, tempo.Spec.Template.Distributor.Replicas),
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
}

func service(tempo v1alpha1.TempoStack) *corev1.Service {
	labels := manifestutils.ComponentLabels(manifestutils.DistributorComponentName, tempo.Name)

	servicePorts := []corev1.ServicePort{
		{
			Name:       manifestutils.OtlpGrpcPortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       manifestutils.PortOtlpGrpcServer,
			TargetPort: intstr.FromString(manifestutils.OtlpGrpcPortName),
		},
		{
			Name:       manifestutils.HttpPortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       manifestutils.PortHTTPServer,
			TargetPort: intstr.FromString(manifestutils.HttpPortName),
		},
	}

	if !tempo.Spec.Template.Gateway.Enabled {
		servicePorts = append(servicePorts, []corev1.ServicePort{
			{
				Name:       manifestutils.PortOtlpHttpName,
				Port:       manifestutils.PortOtlpHttp,
				TargetPort: intstr.FromString(manifestutils.PortOtlpHttpName),
				Protocol:   corev1.ProtocolTCP,
			},
			{
				Name:       manifestutils.PortJaegerThriftHTTPName,
				Port:       manifestutils.PortJaegerThriftHTTP,
				TargetPort: intstr.FromString(manifestutils.PortJaegerThriftHTTPName),
				Protocol:   corev1.ProtocolTCP,
			},
			{
				Name:       manifestutils.PortJaegerThriftCompactName,
				Port:       manifestutils.PortJaegerThriftCompact,
				TargetPort: intstr.FromString(manifestutils.PortJaegerThriftCompactName),
				Protocol:   corev1.ProtocolUDP,
			},
			{
				Name:       manifestutils.PortJaegerThriftBinaryName,
				Port:       manifestutils.PortJaegerThriftBinary,
				TargetPort: intstr.FromString(manifestutils.PortJaegerThriftBinaryName),
				Protocol:   corev1.ProtocolUDP,
			},
			{
				Name:       manifestutils.PortJaegerGrpcName,
				Port:       manifestutils.PortJaegerGrpc,
				TargetPort: intstr.FromString(manifestutils.PortJaegerGrpcName),
				Protocol:   corev1.ProtocolTCP,
			},
			{
				Name:       manifestutils.PortZipkinName,
				Port:       manifestutils.PortZipkin,
				TargetPort: intstr.FromString(manifestutils.PortZipkinName),
				Protocol:   corev1.ProtocolTCP,
			},
		}...)
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.DistributorComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports:    servicePorts,
			Selector: labels,
		},
	}
}

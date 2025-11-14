package monolithic

import (
	"errors"
	"fmt"
	"path"

	"github.com/operator-framework/operator-lib/proxy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/gateway"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

var (
	tenGBQuantity = resource.MustParse("10Gi")
)

// BuildTempoStatefulset creates the Tempo statefulset for a monolithic deployment.
func BuildTempoStatefulset(opts Options, extraAnnotations map[string]string) (*appsv1.StatefulSet, error) {
	tempo := opts.Tempo
	labels := ComponentLabels(manifestutils.TempoMonolithComponentName, tempo.Name)
	annotations := manifestutils.StorageSecretHash(opts.StorageParams, extraAnnotations)
	annotations = manifestutils.AddCertificateHashAnnotations(tempo.GetAnnotations(), annotations)

	sts := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.TempoMonolithComponentName, tempo.Name),
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
					ServiceAccountName: serviceAccountName(tempo),
					NodeSelector:       tempo.Spec.MonolithicSchedulerSpec.NodeSelector,
					Tolerations:        tempo.Spec.MonolithicSchedulerSpec.Tolerations,
					Affinity:           buildAffinity(tempo.Spec.MonolithicSchedulerSpec, labels),
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
							},
							Ports: buildTempoContainerPorts(opts),
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Scheme: corev1.URISchemeHTTP,
										Path:   manifestutils.TempoReadinessPath,
										Port:   intstr.FromString(manifestutils.TempoInternalServerPortName),
									},
								},
								InitialDelaySeconds: 15,
								TimeoutSeconds:      1,
							},
							SecurityContext: manifestutils.TempoContainerSecurityContext(),
							Resources:       ptr.Deref(tempo.Spec.Resources, corev1.ResourceRequirements{}),
						},
					},
					SecurityContext: tempo.Spec.PodSecurityContext,
					Volumes: []corev1.Volume{
						{
							Name: manifestutils.ConfigVolumeName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: naming.Name(manifestutils.TempoConfigName, tempo.Name),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	manifestutils.SetGoMemLimit("tempo", &sts.Spec.Template.Spec)

	err := configureStorage(opts, sts)
	if err != nil {
		return nil, err
	}

	if tempo.Spec.Ingestion != nil && tempo.Spec.Ingestion.OTLP != nil {
		if tempo.Spec.Ingestion.OTLP.GRPC != nil && tempo.Spec.Ingestion.OTLP.GRPC.Enabled &&
			tempo.Spec.Ingestion.OTLP.GRPC.TLS != nil && tempo.Spec.Ingestion.OTLP.GRPC.TLS.Enabled {
			err := manifestutils.MountTLSSpecVolumes(
				&sts.Spec.Template.Spec, "tempo", *tempo.Spec.Ingestion.OTLP.GRPC.TLS,
				manifestutils.ReceiverGRPCTLSCADir, manifestutils.ReceiverGRPCTLSCertDir,
			)
			if err != nil {
				return nil, err
			}
		}

		if tempo.Spec.Ingestion.OTLP.HTTP != nil && tempo.Spec.Ingestion.OTLP.HTTP.Enabled &&
			tempo.Spec.Ingestion.OTLP.HTTP.TLS != nil && tempo.Spec.Ingestion.OTLP.HTTP.TLS.Enabled {
			err := manifestutils.MountTLSSpecVolumes(
				&sts.Spec.Template.Spec, "tempo", *tempo.Spec.Ingestion.OTLP.HTTP.TLS,
				manifestutils.ReceiverHTTPTLSCADir, manifestutils.ReceiverHTTPTLSCertDir,
			)
			if err != nil {
				return nil, err
			}
		}
	}

	if tempo.Spec.JaegerUI != nil && tempo.Spec.JaegerUI.Enabled {
		configureJaegerUI(opts, sts)

	}

	if tempo.Spec.Multitenancy.IsGatewayEnabled() {
		err = configureGateway(opts, sts)

		if opts.CtrlConfig.Gates.HTTPEncryption || opts.CtrlConfig.Gates.GRPCEncryption {
			ids := []int{0}
			for index, container := range sts.Spec.Template.Spec.Containers {
				if container.Name == "tempo-gateway" {
					ids = append(ids, index)
				}

				if container.Name == "jaeger-query" {
					ids = append(ids, index)
				}

				if container.Name == "tempo-query" {
					ids = append(ids, index)
				}
			}

			caBundleName := naming.SigningCABundleName(opts.Tempo.Name)
			if err := manifestutils.ConfigureServiceCA(&sts.Spec.Template.Spec, caBundleName, ids...); err != nil {
				return nil, err
			}

			err := manifestutils.ConfigureServicePKI(opts.Tempo.Name, manifestutils.TempoMonolithComponentName,
				&sts.Spec.Template.Spec, ids...)
			if err != nil {
				return nil, err
			}
		}

		if err != nil {
			return nil, err
		}
	}

	return sts, nil
}

func serviceAccountName(tempo v1alpha1.TempoMonolithic) string {
	if tempo.Spec.ServiceAccount != "" {
		return tempo.Spec.ServiceAccount
	}
	return naming.DefaultServiceAccountName(tempo.Name)
}

func buildAffinity(scheduler v1alpha1.MonolithicSchedulerSpec, labels labels.Set) *corev1.Affinity {
	if scheduler.Affinity != nil {
		return scheduler.Affinity
	}
	return manifestutils.DefaultAffinity(labels)
}

func buildTempoContainerPorts(opts Options) []corev1.ContainerPort {
	tempo := opts.Tempo
	ports := []corev1.ContainerPort{
		{
			Name:          manifestutils.HttpPortName,
			ContainerPort: manifestutils.PortHTTPServer,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          manifestutils.TempoInternalServerPortName,
			ContainerPort: manifestutils.PortInternalHTTPServer,
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
	const volumeName = "tempo-storage"

	if tempo.Spec.Storage == nil {
		return errors.New("storage not configured")
	}

	sts.Spec.Template.Spec.Containers[0].VolumeMounts = append(sts.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
		Name:      volumeName,
		MountPath: "/var/tempo",
	})

	// configure /var/tempo:
	// * memory:         mount /var/tempo on a tmpfs
	// * pv:     		 mount /var/tempo on a Persistent Volume
	// * object storage: also mount /var/tempo on a Persistent Volume, for the WAL
	if tempo.Spec.Storage.Traces.Backend == v1alpha1.MonolithicTracesStorageBackendMemory {
		sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium:    corev1.StorageMediumMemory,
					SizeLimit: tempo.Spec.Storage.Traces.Size,
				},
			},
		})
	} else {
		// object storage also needs a PVC to store the WAL
		sts.Spec.VolumeClaimTemplates = append(sts.Spec.VolumeClaimTemplates, corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: ptr.Deref(tempo.Spec.Storage.Traces.Size, tenGBQuantity),
					},
				},
				StorageClassName: tempo.Spec.Storage.Traces.StorageClassName,
				VolumeMode:       ptr.To(corev1.PersistentVolumeFilesystem),
			},
		})
	}

	switch tempo.Spec.Storage.Traces.Backend {
	case v1alpha1.MonolithicTracesStorageBackendMemory, v1alpha1.MonolithicTracesStorageBackendPV:
		// for memory and PV storage, /var/tempo is configured above.

	case v1alpha1.MonolithicTracesStorageBackendS3:
		if tempo.Spec.Storage.Traces.S3 == nil {
			return errors.New("please configure .spec.storage.traces.s3")
		}

		err := manifestutils.ConfigureS3Storage(&sts.Spec.Template.Spec,
			"tempo", tempo.Spec.Storage.Traces.S3.Secret,
			tempo.Spec.Storage.Traces.S3.TLS, opts.StorageParams.CredentialMode, tempo.Name, opts.StorageParams.CloudCredentials.Environment)
		if err != nil {
			return err
		}

	case v1alpha1.MonolithicTracesStorageBackendAzure:
		if tempo.Spec.Storage.Traces.Azure == nil {
			return errors.New("please configure .spec.storage.traces.azure")
		}
		err := manifestutils.ConfigureAzureStorage(&sts.Spec.Template.Spec, opts.StorageParams.AzureStorage, "tempo",
			tempo.Spec.Storage.Traces.Azure.Secret, opts.StorageParams.CredentialMode)
		if err != nil {
			return err
		}

	case v1alpha1.MonolithicTracesStorageBackendGCS:
		if tempo.Spec.Storage.Traces.GCS == nil {
			return errors.New("please configure .spec.storage.traces.gcs")
		}

		err := manifestutils.ConfigureGCS(&sts.Spec.Template.Spec, "tempo",
			tempo.Spec.Storage.Traces.GCS.Secret, opts.StorageParams.GCS.Audience, opts.StorageParams.CredentialMode)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("invalid storage backend: '%s'", tempo.Spec.Storage.Traces.Backend)
	}
	return nil
}

func configureJaegerUI(opts Options, sts *appsv1.StatefulSet) {
	const tmpVolumeName = "tempo-query-tmp"
	tempo := opts.Tempo

	args := []string{
		"--query.base-path=/",
		"--span-storage.type=grpc",
		"--grpc-storage.server=localhost:7777",
		"--query.bearer-token-propagation=true",
	}

	// multi-tenancy enabled, possibly without gateway. forward X-Scope-OrgID header.
	if tempo.Spec.Multitenancy != nil && tempo.Spec.Multitenancy.Enabled {
		args = append(args, []string{
			"--multi-tenancy.enabled=true",
			fmt.Sprintf("--multi-tenancy.header=%s", manifestutils.TenantHeader),
		}...)
	}

	// all connections to Jaeger UI must go via gateway
	if tempo.Spec.Multitenancy.IsGatewayEnabled() {
		args = append(args, []string{
			fmt.Sprintf("--query.grpc-server.host-port=localhost:%d", manifestutils.PortJaegerGRPCQuery),
			fmt.Sprintf("--query.http-server.host-port=localhost:%d", manifestutils.PortJaegerUI),
			fmt.Sprintf("--admin.http.host-port=localhost:%d", manifestutils.PortJaegerMetrics),
		}...)
	}

	jaegerQueryContainer := corev1.Container{
		Name:  "jaeger-query",
		Image: opts.CtrlConfig.DefaultImages.JaegerQuery,
		Env:   proxy.ReadProxyVarsFromEnv(),
		Args:  args,
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
				Name:      tmpVolumeName,
				MountPath: "/tmp",
			},
		},
		SecurityContext: manifestutils.TempoContainerSecurityContext(),
		Resources:       ptr.Deref(opts.Tempo.Spec.JaegerUI.Resources, corev1.ResourceRequirements{}),
	}

	tempoQuery := corev1.Container{
		Name:  "tempo-query",
		Image: opts.CtrlConfig.DefaultImages.TempoQuery,
		Env:   proxy.ReadProxyVarsFromEnv(),
		Args: []string{
			"-config=/conf/tempo-query.yaml",
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          manifestutils.TempoGRPCQuery,
				ContainerPort: manifestutils.PortTempoGRPCQuery,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Resources: ptr.Deref(opts.Tempo.Spec.JaegerUI.TempoQueryResources, corev1.ResourceRequirements{}),
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      manifestutils.ConfigVolumeName,
				MountPath: "/conf",
				ReadOnly:  true,
			},
		},
		SecurityContext: manifestutils.TempoContainerSecurityContext(),
	}

	tempoQueryVolume := corev1.Volume{
		Name: tmpVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, jaegerQueryContainer)
	sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, tempoQuery)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, tempoQueryVolume)

	manifestutils.SetGoMemLimit("jaeger-query", &sts.Spec.Template.Spec)
	manifestutils.SetGoMemLimit("tempo-query", &sts.Spec.Template.Spec)

}

func configureGateway(opts Options, sts *appsv1.StatefulSet) error {
	var (
		containerName   = "tempo-gateway"
		gatewayMountDir = "/etc/tempo-gateway"
		servingCADir    = path.Join(gatewayMountDir, "serving-ca")
		servingCertDir  = path.Join(gatewayMountDir, "serving-cert")
	)

	tempo := opts.Tempo
	args := []string{
		fmt.Sprintf("--web.listen=0.0.0.0:%d", manifestutils.GatewayPortHTTPServer),                  // proxies Tempo API and optionally Jaeger UI
		fmt.Sprintf("--web.internal.listen=0.0.0.0:%d", manifestutils.GatewayPortInternalHTTPServer), // serves health checks
		fmt.Sprintf("--traces.tenant-header=%s", manifestutils.TenantHeader),
		fmt.Sprintf("--traces.tempo.endpoint=%s://localhost:%d", gateway.HttpScheme(opts.CtrlConfig.Gates.HTTPEncryption), manifestutils.PortHTTPServer), // Tempo API upstream
		fmt.Sprintf("--traces.write-timeout=%s", opts.Tempo.Spec.Timeout.Duration.String()),
		fmt.Sprintf("--rbac.config=%s", path.Join(gatewayMountDir, "rbac", manifestutils.GatewayRBACFileName)),
		fmt.Sprintf("--tenants.config=%s", path.Join(gatewayMountDir, "tenants", manifestutils.GatewayTenantFileName)),
		"--log.level=info",
	}

	if opts.CtrlConfig.Gates.HTTPEncryption {
		args = append(args, []string{
			fmt.Sprintf("--tls.internal.server.key-file=%s", path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSKeyFilename)),
			fmt.Sprintf("--tls.internal.server.cert-file=%s", path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSCertFilename)),
			fmt.Sprintf("--traces.tls.key-file=%s", path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSKeyFilename)),
			fmt.Sprintf("--traces.tls.cert-file=%s", path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSCertFilename)),
			fmt.Sprintf("--traces.tls.ca-file=%s", path.Join(manifestutils.TempoInternalTLSCADir, manifestutils.TLSCAFilename)),
			"--traces.tls.watch-certs=true",
		}...)
	}

	ports := []corev1.ContainerPort{
		{
			Name:          manifestutils.GatewayHttpPortName,
			ContainerPort: manifestutils.GatewayPortHTTPServer,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          manifestutils.GatewayInternalHttpPortName,
			ContainerPort: manifestutils.GatewayPortInternalHTTPServer,
			Protocol:      corev1.ProtocolTCP,
		},
	}

	if tempo.Spec.Ingestion != nil && tempo.Spec.Ingestion.OTLP != nil && tempo.Spec.Ingestion.OTLP.GRPC != nil && tempo.Spec.Ingestion.OTLP.GRPC.Enabled {
		args = append(args, fmt.Sprintf("--grpc.listen=0.0.0.0:%d", manifestutils.GatewayPortGRPCServer))                    // proxies Tempo Distributor gRPC
		args = append(args, fmt.Sprintf("--traces.write.otlpgrpc.endpoint=localhost:%d", manifestutils.PortOtlpGrpcServer))  // Tempo Distributor gRPC upstream
		args = append(args, fmt.Sprintf("--traces.write.otlphttp.endpoint=http://localhost:%d", manifestutils.PortOtlpHttp)) // Tempo Distributor HTTP upstream
		ports = append(ports, corev1.ContainerPort{
			Name:          manifestutils.GatewayGrpcPortName,
			ContainerPort: manifestutils.GatewayPortGRPCServer,
			Protocol:      corev1.ProtocolTCP,
		})
	}

	if tempo.Spec.JaegerUI != nil && tempo.Spec.JaegerUI.Enabled {
		args = append(args, fmt.Sprintf("--traces.read.endpoint=http://localhost:%d", manifestutils.PortJaegerQuery)) // Jaeger UI upstream
	}

	if tempo.Spec.Query != nil && tempo.Spec.Query.RBAC.Enabled {
		args = append(args, "--traces.query-rbac=true")
	}

	if opts.CtrlConfig.Gates.OpenShift.ServingCertsService {
		args = append(args, []string{
			fmt.Sprintf("--tls.server.cert-file=%s", path.Join(servingCertDir, "tls.crt")), // TLS of public HTTP (8080) and gRPC (8090) server
			fmt.Sprintf("--tls.server.key-file=%s", path.Join(servingCertDir, "tls.key")),
			fmt.Sprintf("--tls.healthchecks.server-ca-file=%s", path.Join(servingCADir, "service-ca.crt")),
			fmt.Sprintf("--tls.healthchecks.server-name=%s", naming.ServiceFqdn(tempo.Namespace, tempo.Name, manifestutils.GatewayComponentName)),
			"--web.healthchecks.url=https://localhost:8080",
			"--tls.client-auth-type=NoClientCert",
		}...)
	}

	gatewayContainer := corev1.Container{
		Name:           containerName,
		Image:          opts.CtrlConfig.DefaultImages.TempoGateway,
		Env:            proxy.ReadProxyVarsFromEnv(),
		Args:           args,
		Ports:          ports,
		LivenessProbe:  gateway.LivenessProbe(opts.CtrlConfig.Gates.HTTPEncryption),
		ReadinessProbe: gateway.ReadinessProbe(opts.CtrlConfig.Gates.HTTPEncryption),
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "gateway-rbac",
				ReadOnly:  true,
				MountPath: path.Join(gatewayMountDir, "rbac"),
			},
			{
				Name:      "gateway-tenants",
				ReadOnly:  true,
				MountPath: path.Join(gatewayMountDir, "tenants"),
			},
		},
		Resources:       ptr.Deref(opts.Tempo.Spec.Multitenancy.Resources, corev1.ResourceRequirements{}),
		SecurityContext: manifestutils.TempoContainerSecurityContext(),
	}

	gatewayVolumes := []corev1.Volume{
		{
			Name: "gateway-rbac",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: naming.Name(manifestutils.GatewayComponentName, tempo.Name),
					},
					Items: []corev1.KeyToPath{
						{
							Key:  manifestutils.GatewayRBACFileName,
							Path: manifestutils.GatewayRBACFileName,
						},
					},
				},
			},
		},
		{
			Name: "gateway-tenants",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: naming.Name(manifestutils.GatewayComponentName, tempo.Name),
					Items: []corev1.KeyToPath{
						{
							Key:  manifestutils.GatewayTenantFileName,
							Path: manifestutils.GatewayTenantFileName,
						},
					},
				},
			},
		},
	}

	sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, gatewayContainer)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, gatewayVolumes...)

	if tempo.Spec.Multitenancy.TenantsSpec.Mode == v1alpha1.ModeOpenShift {
		opaContainer := gateway.NewOpaContainer(
			opts.CtrlConfig,
			tempo.Spec.Multitenancy.TenantsSpec,
			tempo.Spec.Query.RBAC.Enabled,
			opaPackage,
			ptr.Deref(opts.Tempo.Spec.Multitenancy.Resources, corev1.ResourceRequirements{}),
		)
		sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, opaContainer)
	}

	if opts.CtrlConfig.Gates.OpenShift.ServingCertsService {
		err := manifestutils.MountCAConfigMap(&sts.Spec.Template.Spec, containerName, naming.ServingCABundleName(tempo.Name), servingCADir)
		if err != nil {
			return err
		}

		err = manifestutils.MountCertSecret(&sts.Spec.Template.Spec, containerName, naming.ServingCertName(manifestutils.GatewayComponentName, tempo.Name), servingCertDir)
		if err != nil {
			return err
		}
	}

	return nil
}

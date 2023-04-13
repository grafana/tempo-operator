package queryfrontend

import (
	"fmt"

	routev1 "github.com/openshift/api/route/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
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
	grpclbPortName        = "grpclb"
	jaegerMetricsPortName = "jaeger-metrics"
	jaegerUIPortName      = "jaeger-ui"
	portGRPCLBServer      = 9096
	portJaegerUI          = 16686
	portJaegerMetrics     = 16687
)

// BuildQueryFrontend creates the query-frontend objects.
func BuildQueryFrontend(params manifestutils.Params) ([]client.Object, error) {
	var manifests []client.Object

	d, err := deployment(params)
	if err != nil {
		return nil, err
	}

	d.Spec.Template, err = manifestutils.PatchTracingJaegerEnv(params.Tempo, d.Spec.Template)
	if err != nil {
		return nil, err
	}
	gates := params.Gates
	tempo := params.Tempo

	if gates.HTTPEncryption || gates.GRPCEncryption {
		caBundleName := naming.SigningCABundleName(tempo.Name)
		if err := manifestutils.ConfigureServiceCA(&d.Spec.Template.Spec, caBundleName, 0, 1); err != nil {
			return nil, err
		}

		err := manifestutils.ConfigureServicePKI(tempo.Name, manifestutils.QueryFrontendComponentName, &d.Spec.Template.Spec, 0, 1)
		if err != nil {
			return nil, err
		}
	}

	manifests = append(manifests, d)

	svcs := services(tempo)
	for _, s := range svcs {
		manifests = append(manifests, s)
	}

	if !tempo.Spec.Template.Gateway.Enabled {
		//exhaustive:ignore
		switch tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Type {
		case v1alpha1.IngressTypeIngress:
			manifests = append(manifests, ingress(tempo))
		case v1alpha1.IngressTypeRoute:
			manifests = append(manifests, route(tempo))
		}
	}

	return manifests, nil
}

func deployment(params manifestutils.Params) (*v1.Deployment, error) {
	tempo := params.Tempo
	labels := manifestutils.ComponentLabels(manifestutils.QueryFrontendComponentName, tempo.Name)
	annotations := manifestutils.CommonAnnotations(params.ConfigChecksum)
	cfg := tempo.Spec.Template.QueryFrontend

	d := &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: v1.DeploymentSpec{
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
							Args: []string{
								"-target=query-frontend",
								"-config.file=/conf/tempo.yaml",
								"-mem-ballast-size-mbs=1024",
							},
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
							},
							ReadinessProbe: manifestutils.TempoReadinessProbe(params.Gates.HTTPEncryption || params.Gates.GRPCEncryption),
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
							Resources:       manifestutils.Resources(tempo, manifestutils.QueryFrontendComponentName),
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

	if tempo.Spec.Template.QueryFrontend.JaegerQuery.Enabled {
		jaegerQueryContainer := corev1.Container{
			Name:  "tempo-query",
			Image: tempo.Spec.Images.TempoQuery,
			Args: []string{
				"--query.base-path=/",
				"--grpc-storage-plugin.configuration-file=/conf/tempo-query.yaml",
				"--query.bearer-token-propagation=true",
			},
			Ports: []corev1.ContainerPort{
				{
					Name:          jaegerUIPortName,
					ContainerPort: portJaegerUI,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          jaegerMetricsPortName,
					ContainerPort: portJaegerMetrics,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      manifestutils.ConfigVolumeName,
					MountPath: "/conf",
					ReadOnly:  true,
				},
				{
					Name:      manifestutils.TmpStorageVolumeName + "-query",
					MountPath: manifestutils.TmpStoragePath,
				},
			},
			Resources: manifestutils.Resources(tempo, manifestutils.QueryFrontendComponentName),
		}
		jaegerQueryVolume := corev1.Volume{
			Name: manifestutils.TmpStorageVolumeName + "-query",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}

		// TODO it should be possible to enable multitenancy just for tempo, without the gateway
		if tempo.Spec.Tenants != nil {
			jaegerQueryContainer.Args = append(jaegerQueryContainer.Args, []string{
				"--multi-tenancy.enabled=true",
				fmt.Sprintf("--multi-tenancy.header=%s", manifestutils.TenantHeader),
			}...)
		}

		d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers, jaegerQueryContainer)
		d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, jaegerQueryVolume)
	}

	err := manifestutils.ConfigureStorage(tempo, &d.Spec.Template.Spec)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func services(tempo v1alpha1.TempoStack) []*corev1.Service {
	labels := manifestutils.ComponentLabels(manifestutils.QueryFrontendComponentName, tempo.Name)
	frontEndService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       manifestutils.HttpPortName,
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

	queryFrontendDiscoveryName := manifestutils.QueryFrontendComponentName + "-discovery"
	frontEndDiscoveryService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(queryFrontendDiscoveryName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    manifestutils.ComponentLabels(queryFrontendDiscoveryName, tempo.Name),
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			// We set PublishNotReadyAddresses to true so that the service always returns the entire list
			// of A records for matching pods, irrespective if they are in Ready state or not.
			// This is especially useful during startup of query-frontend and querier, where query-frontend
			// only gets Ready if at least one querier connects to it (and without this setting, querier could
			// never connect to query-frontend-discovery-svc because it would not return A records of not-ready pods).
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name:       manifestutils.HttpPortName,
					Port:       manifestutils.PortHTTPServer,
					TargetPort: intstr.FromString(manifestutils.HttpPortName),
				},
				{
					Name:       manifestutils.GrpcPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortGRPCServer,
					TargetPort: intstr.FromString(manifestutils.GrpcPortName),
				},
				{
					Name:       grpclbPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       portGRPCLBServer,
					TargetPort: intstr.FromString(grpclbPortName),
				},
			},
			Selector: labels,
		},
	}

	if tempo.Spec.Template.QueryFrontend.JaegerQuery.Enabled {
		jaegerPorts := []corev1.ServicePort{
			{
				Name:       jaegerUIPortName,
				Port:       portJaegerUI,
				TargetPort: intstr.FromString(jaegerUIPortName),
			},
			{
				Name:       jaegerMetricsPortName,
				Port:       portJaegerMetrics,
				TargetPort: intstr.FromString(jaegerMetricsPortName),
			},
		}

		frontEndService.Spec.Ports = append(frontEndService.Spec.Ports, jaegerPorts...)
		frontEndDiscoveryService.Spec.Ports = append(frontEndDiscoveryService.Spec.Ports, jaegerPorts...)
	}

	return []*corev1.Service{frontEndService, frontEndDiscoveryService}
}

func ingress(tempo v1alpha1.TempoStack) *networkingv1.Ingress {
	queryFrontendName := naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name)
	labels := manifestutils.ComponentLabels(manifestutils.QueryFrontendComponentName, tempo.Name)

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        queryFrontendName,
			Namespace:   tempo.Namespace,
			Labels:      labels,
			Annotations: tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Annotations,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.IngressClassName,
		},
	}

	backend := networkingv1.IngressBackend{
		Service: &networkingv1.IngressServiceBackend{
			Name: queryFrontendName,
			Port: networkingv1.ServiceBackendPort{
				Name: jaegerUIPortName,
			},
		},
	}

	if tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Host == "" {
		ingress.Spec.DefaultBackend = &backend
	} else {
		pathType := networkingv1.PathTypePrefix
		ingress.Spec.Rules = []networkingv1.IngressRule{
			{
				Host: tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Host,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathType,
								Backend:  backend,
							},
						},
					},
				},
			},
		}
	}

	return ingress
}

func route(tempo v1alpha1.TempoStack) *routev1.Route {
	queryFrontendName := naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name)
	labels := manifestutils.ComponentLabels(manifestutils.QueryFrontendComponentName, tempo.Name)

	var tlsCfg *routev1.TLSConfig
	switch tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Route.Termination {
	case v1alpha1.TLSRouteTerminationTypeInsecure:
		// NOTE: insecure, no tls cfg.
	case v1alpha1.TLSRouteTerminationTypeEdge:
		tlsCfg = &routev1.TLSConfig{Termination: routev1.TLSTerminationEdge}
	case v1alpha1.TLSRouteTerminationTypePassthrough:
		tlsCfg = &routev1.TLSConfig{Termination: routev1.TLSTerminationPassthrough}
	case v1alpha1.TLSRouteTerminationTypeReencrypt:
		tlsCfg = &routev1.TLSConfig{Termination: routev1.TLSTerminationReencrypt}
	default: // NOTE: if unsupported, end here.
		return nil
	}

	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:        queryFrontendName,
			Namespace:   tempo.Namespace,
			Labels:      labels,
			Annotations: tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Annotations,
		},
		Spec: routev1.RouteSpec{
			Host: tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Host,
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: queryFrontendName,
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString(jaegerUIPortName),
			},
			TLS: tlsCfg,
		},
	}
}

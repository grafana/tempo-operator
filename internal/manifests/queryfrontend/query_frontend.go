package queryfrontend

import (
	"fmt"
	"path"
	"strings"

	"github.com/imdario/mergo"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/operator-framework/operator-lib/proxy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/memberlist"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
	"github.com/grafana/tempo-operator/internal/manifests/oauthproxy"
)

const (
	grpclbPortName                   = "grpclb"
	portGRPCLBServer                 = 9096
	thanosQuerierOpenShiftMonitoring = "https://thanos-querier.openshift-monitoring.svc.cluster.local:9091"
)

const (
	containerNameTempo       = "tempo"
	containerNameJaegerQuery = "jaeger-query"
	containerNameTempoQuery  = "tempo-query"

	timeoutRouteAnnotation = "haproxy.router.openshift.io/timeout"
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
	gates := params.CtrlConfig.Gates
	tempo := params.Tempo

	if gates.HTTPEncryption || gates.GRPCEncryption {
		caBundleName := naming.SigningCABundleName(tempo.Name)
		targets := []string{containerNameTempo, containerNameJaegerQuery, containerNameTempoQuery}
		if err := manifestutils.ConfigureServiceCAByContainerName(&d.Spec.Template.Spec, caBundleName, targets...); err != nil {
			return nil, err
		}

		err := manifestutils.ConfigureServicePKIByContainerName(tempo.Name, manifestutils.QueryFrontendComponentName, &d.Spec.Template.Spec, targets...)
		if err != nil {
			return nil, err
		}
	}

	svcs := services(params)
	for _, s := range svcs {
		manifests = append(manifests, s)
	}

	if !tempo.Spec.Template.Gateway.Enabled {
		//exhaustive:ignore
		switch tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Type {
		case v1alpha1.IngressTypeIngress:
			manifests = append(manifests, ingress(tempo))
		case v1alpha1.IngressTypeRoute:
			routeObj, err := route(tempo)
			if err != nil {
				return nil, err
			}

			jaegerUIAuthentication := tempo.Spec.Template.QueryFrontend.JaegerQuery.Authentication

			if jaegerUIAuthentication != nil && jaegerUIAuthentication.Enabled {
				oauthproxy.PatchDeploymentForOauthProxy(
					tempo.ObjectMeta,
					params.CtrlConfig,
					tempo.Spec.Template.QueryFrontend.JaegerQuery.Authentication,
					tempo.Spec.Timeout.Duration,
					tempo.Spec.Images,
					d)

				oauthproxy.PatchQueryFrontEndService(getQueryFrontendService(tempo, svcs), tempo.Name)
				manifests = append(manifests, oauthproxy.OAuthServiceAccount(params))
				oauthproxy.PatchRouteForOauthProxy(routeObj)
			}
			manifests = append(manifests, routeObj)
		}
	}

	manifests = append(manifests, d)

	if tempo.Spec.Template.QueryFrontend.JaegerQuery.Enabled && tempo.Spec.Template.QueryFrontend.JaegerQuery.MonitorTab.Enabled &&
		tempo.Spec.Template.QueryFrontend.JaegerQuery.MonitorTab.PrometheusEndpoint == thanosQuerierOpenShiftMonitoring {
		clusterRoleBinding := openShiftMonitoringClusterRoleBinding(tempo, d)
		manifests = append(manifests, &clusterRoleBinding)
	}

	return manifests, nil
}

func getQueryFrontendService(tempo v1alpha1.TempoStack, services []*corev1.Service) *corev1.Service {
	serviceName := naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name)
	for _, svc := range services {
		if svc.Name == serviceName {
			return svc
		}
	}
	return nil
}

func resources(tempo v1alpha1.TempoStack) corev1.ResourceRequirements {
	if tempo.Spec.Template.QueryFrontend.Resources == nil {
		return manifestutils.Resources(tempo, manifestutils.QueryFrontendComponentName, tempo.Spec.Template.QueryFrontend.Replicas)
	}
	return *tempo.Spec.Template.QueryFrontend.Resources
}

func tempoQueryResources(tempo v1alpha1.TempoStack) corev1.ResourceRequirements {
	if tempo.Spec.Template.QueryFrontend.JaegerQuery.TempoQuery.Resources == nil {
		return manifestutils.Resources(tempo, manifestutils.QueryFrontendComponentName, tempo.Spec.Template.QueryFrontend.Replicas)
	}
	return *tempo.Spec.Template.QueryFrontend.JaegerQuery.Resources
}

func jaegerQueryResources(tempo v1alpha1.TempoStack) corev1.ResourceRequirements {
	if tempo.Spec.Template.QueryFrontend.JaegerQuery.Resources == nil {
		return manifestutils.Resources(tempo, manifestutils.JaegerFrontendComponentName, tempo.Spec.Template.QueryFrontend.Replicas)
	}
	return *tempo.Spec.Template.QueryFrontend.JaegerQuery.Resources
}

func deployment(params manifestutils.Params) (*appsv1.Deployment, error) {
	tempo := params.Tempo
	labels := manifestutils.ComponentLabels(manifestutils.QueryFrontendComponentName, tempo.Name)
	annotations := manifestutils.CommonAnnotations(params.ConfigChecksum)
	cfg := tempo.Spec.Template.QueryFrontend
	tempoImage := tempo.Spec.Images.Tempo
	if tempoImage == "" {
		tempoImage = params.CtrlConfig.DefaultImages.Tempo
	}
	jaegerQueryImage := tempo.Spec.Images.JaegerQuery
	if jaegerQueryImage == "" {
		jaegerQueryImage = params.CtrlConfig.DefaultImages.JaegerQuery
	}

	tempoQueryImage := tempo.Spec.Images.TempoQuery
	if tempoQueryImage == "" {
		tempoQueryImage = params.CtrlConfig.DefaultImages.TempoQuery
	}

	d := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: tempo.Spec.Template.QueryFrontend.Replicas,
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
							Name:  containerNameTempo,
							Image: tempoImage,
							Env:   proxy.ReadProxyVarsFromEnv(),
							Args: []string{
								"-target=query-frontend",
								"-config.file=/conf/tempo-query-frontend.yaml",
								"-mem-ballast-size-mbs=1024",
								"-log.level=info",
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
							ReadinessProbe: manifestutils.TempoReadinessProbe(params.CtrlConfig.Gates.HTTPEncryption && params.Tempo.Spec.Template.Gateway.Enabled),
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
					SecurityContext: tempo.Spec.Template.QueryFrontend.PodSecurityContext,
				},
			},
		},
	}

	if tempo.Spec.Template.QueryFrontend.JaegerQuery.Enabled {
		jaegerQueryContainer := corev1.Container{
			Name:  containerNameJaegerQuery,
			Image: jaegerQueryImage,
			Env:   proxy.ReadProxyVarsFromEnv(),
			Args: []string{
				"--query.base-path=/",
				"--span-storage.type=grpc",
				"--grpc-storage.server=localhost:7777",
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
					Name:      manifestutils.TmpStorageVolumeName + "-query",
					MountPath: manifestutils.TmpStoragePath,
				},
			},
			Resources:       jaegerQueryResources(tempo),
			SecurityContext: manifestutils.TempoContainerSecurityContext(),
		}

		tempoProxyContainer := corev1.Container{
			Name:  containerNameTempoQuery,
			Image: tempoQueryImage,
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
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      manifestutils.ConfigVolumeName,
					MountPath: "/conf",
					ReadOnly:  true,
				},
			},
			Resources:       tempoQueryResources(tempo),
			SecurityContext: manifestutils.TempoContainerSecurityContext(),
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					GRPC: &corev1.GRPCAction{
						Port: manifestutils.PortTempoGRPCQuery,
					},
				},
				TimeoutSeconds:   1,
				PeriodSeconds:    5,
				FailureThreshold: 12,
			},
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

		if params.CtrlConfig.Gates.HTTPEncryption && tempo.Spec.Template.Gateway.Enabled {
			jaegerQueryContainer.Args = append(jaegerQueryContainer.Args,
				"--query.http.tls.enabled=true",
				fmt.Sprintf("--query.http.tls.key=%s", path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSKeyFilename)),
				fmt.Sprintf("--query.http.tls.cert=%s", path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSCertFilename)),
				fmt.Sprintf("--query.http.tls.client-ca=%s", path.Join(manifestutils.TempoInternalTLSCADir, manifestutils.TLSCAFilename)),
			)
		}

		if params.CtrlConfig.Gates.GRPCEncryption && tempo.Spec.Template.Gateway.Enabled {
			jaegerQueryContainer.Args = append(jaegerQueryContainer.Args,
				"--query.grpc.tls.enabled=true",
				fmt.Sprintf("--query.grpc.tls.key=%s", path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSKeyFilename)),
				fmt.Sprintf("--query.grpc.tls.cert=%s", path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSCertFilename)),
				fmt.Sprintf("--query.grpc.tls.client-ca=%s", path.Join(manifestutils.TempoInternalTLSCADir, manifestutils.TLSCAFilename)),
			)
		}

		if tempo.Spec.Template.QueryFrontend.JaegerQuery.MonitorTab.Enabled {
			c, err := enableMonitoringTab(tempo, jaegerQueryContainer)
			if err != nil {
				return nil, fmt.Errorf("failed to configure monitor tab in tempo-query container: %w", err)
			}
			jaegerQueryContainer = c
		}

		d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers, jaegerQueryContainer)
		d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers, tempoProxyContainer)
		d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, jaegerQueryVolume)
	}

	err := manifestutils.ConfigureStorage(params.StorageParams, tempo, &d.Spec.Template.Spec, "tempo")
	if err != nil {
		return nil, err
	}
	return d, nil
}

func enableMonitoringTab(tempo v1alpha1.TempoStack, jaegerQueryContainer corev1.Container) (corev1.Container, error) {
	// TODO (pavolloffay) disable/enable monitoring tab https://github.com/grafana/tempo-operator/issues/464
	container := corev1.Container{
		Env: []corev1.EnvVar{
			{
				Name:  "METRICS_STORAGE_TYPE",
				Value: "prometheus",
			},
			{
				Name:  "PROMETHEUS_SERVER_URL",
				Value: tempo.Spec.Template.QueryFrontend.JaegerQuery.MonitorTab.PrometheusEndpoint,
			},
		},
		Args: []string{
			// Just a note that normalization needs to be enabled for < 0.80.0 OTEL collector versions
			// However, we do not intend to support them.
			// --prometheus.query.normalize-calls
			// --prometheus.query.normalize-duration
		},
	}
	// If the endpoint matches Prometheus on OpenShift, configure TLS and token based auth
	prometheusEndpoint := strings.TrimSpace(tempo.Spec.Template.QueryFrontend.JaegerQuery.MonitorTab.PrometheusEndpoint)
	if prometheusEndpoint == thanosQuerierOpenShiftMonitoring {
		container.Args = append(container.Args,
			"--prometheus.tls.enabled=true",
			// This enables token propagation, however flag --query.bearer-token-propagation=true
			// enabled bearer token propagation, overrides the settings and token from the context (incoming) request is used.
			"--prometheus.token-file=/var/run/secrets/kubernetes.io/serviceaccount/token",
			"--prometheus.token-override-from-context=false",
			"--prometheus.tls.ca=/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt")
	}

	err := mergo.Merge(&jaegerQueryContainer, container, mergo.WithAppendSlice)
	if err != nil {
		return corev1.Container{}, err
	}
	return jaegerQueryContainer, nil
}

func openShiftMonitoringClusterRoleBinding(tempo v1alpha1.TempoStack, d *appsv1.Deployment) rbacv1.ClusterRoleBinding {
	labels := manifestutils.ComponentLabels(manifestutils.QueryFrontendComponentName, tempo.Name)
	return rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   naming.Name("cluster-monitoring-view", tempo.Name),
			Labels: labels,
		},
		Subjects: []rbacv1.Subject{
			{
				Name:      d.Spec.Template.Spec.ServiceAccountName,
				Kind:      "ServiceAccount",
				Namespace: tempo.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-monitoring-view",
		},
	}
}

func services(params manifestutils.Params) []*corev1.Service {
	tempo := params.Tempo
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
				Name:       manifestutils.JaegerGRPCQuery,
				Port:       manifestutils.PortJaegerGRPCQuery,
				TargetPort: intstr.FromString(manifestutils.JaegerGRPCQuery),
			},
			{
				Name:       manifestutils.JaegerUIPortName,
				Port:       int32(manifestutils.PortJaegerUI),
				TargetPort: intstr.FromString(manifestutils.JaegerUIPortName),
			},
			{
				Name:       manifestutils.JaegerMetricsPortName,
				Port:       manifestutils.PortJaegerMetrics,
				TargetPort: intstr.FromString(manifestutils.JaegerMetricsPortName),
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
				Name: manifestutils.JaegerUIPortName,
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

func route(tempo v1alpha1.TempoStack) (*routev1.Route, error) {
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
		return nil, fmt.Errorf("unsupported tls termination specified for route")
	}

	serviceName := naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name)

	annotations := tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}
	if annotations[timeoutRouteAnnotation] == "" {
		annotations[timeoutRouteAnnotation] = fmt.Sprintf("%ds", int(tempo.Spec.Timeout.Duration.Seconds()))
	}

	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:        queryFrontendName,
			Namespace:   tempo.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: routev1.RouteSpec{
			Host: tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Host,
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: serviceName,
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString(manifestutils.JaegerUIPortName),
			},
			TLS: tlsCfg,
		},
	}, nil
}

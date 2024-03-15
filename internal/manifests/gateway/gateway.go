package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"path"

	"github.com/imdario/mergo"
	"github.com/operator-framework/operator-lib/proxy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

const (
	// tempoGatewayMountDir is the path that is mounted from the configmap.
	tempoGatewayMountDir = "/etc/tempo-gateway"
)

// BuildGateway creates gateway objects.
func BuildGateway(params manifestutils.Params) ([]client.Object, error) {
	tempo := params.Tempo
	labels := manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name)
	gatewayObjectName := naming.Name(manifestutils.GatewayComponentName, tempo.Name)
	cfgOpts := NewConfigOptions(
		tempo.Namespace,
		tempo.Name,
		gatewayObjectName,
		naming.RouteFqdn(tempo.Namespace, tempo.Name, manifestutils.GatewayComponentName, params.CtrlConfig.Gates.OpenShift.BaseDomain),
		"tempostack",
		*tempo.Spec.Tenants,
		params.GatewayTenantSecret,
		params.GatewayTenantsData,
	)

	rbacConfigMap, rbacCfgHash, err := NewRBACConfigMap(cfgOpts, tempo.Namespace, gatewayObjectName, labels)
	if err != nil {
		return nil, err
	}

	tenantsSecret, tenantsCfgHash, err := NewTenantsSecret(cfgOpts, tempo.Namespace, gatewayObjectName, labels)
	if err != nil {
		return nil, err
	}

	objs := []client.Object{
		rbacConfigMap,
		tenantsSecret,
		service(params.Tempo, params.CtrlConfig.Gates.OpenShift.ServingCertsService),
	}

	dep := deployment(params, rbacCfgHash, tenantsCfgHash)

	if params.CtrlConfig.Gates.HTTPEncryption || params.CtrlConfig.Gates.GRPCEncryption {
		caBundleName := naming.SigningCABundleName(params.Tempo.Name)
		if err := manifestutils.ConfigureServiceCA(&dep.Spec.Template.Spec, caBundleName); err != nil {
			return nil, err
		}
		err := manifestutils.ConfigureServicePKI(params.Tempo.Name, manifestutils.GatewayComponentName, &dep.Spec.Template.Spec)
		if err != nil {
			return nil, err
		}
	}

	if params.Tempo.Spec.Tenants.Mode == v1alpha1.ModeOpenShift {
		dep = patchOCPServiceAccount(params.Tempo, dep)
		dep, err = patchOCPOPAContainer(params, dep)
		if err != nil {
			return nil, err
		}

		objs = append(objs, []client.Object{
			NewAccessReviewClusterRole(
				gatewayObjectName,
				manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name),
			),
			NewAccessReviewClusterRoleBinding(
				gatewayObjectName,
				manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name),
				tempo.Namespace,
				gatewayObjectName,
			),
			serviceAccount(params.Tempo),
		}...)

		if params.CtrlConfig.Gates.OpenShift.ServingCertsService {
			objs = append(objs, manifestutils.NewConfigMapCABundle(
				tempo.Namespace,
				naming.Name("gateway-cabundle", tempo.Name),
				manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name),
			))

			dep, err = patchOCPServingCerts(params.Tempo, dep)
			if err != nil {
				return nil, err
			}
		}
	}

	if params.Tempo.Spec.Template.Gateway.Ingress.Type == v1alpha1.IngressTypeIngress {
		objs = append(objs, ingress(params.Tempo))
	} else if params.Tempo.Spec.Template.Gateway.Ingress.Type == v1alpha1.IngressTypeRoute {
		routeObj, err := route(params.Tempo)
		if err != nil {
			return nil, err
		}
		objs = append(objs, routeObj)
	}

	dep.Spec.Template, err = patchTraceReadEndpoint(params, dep.Spec.Template)
	if err != nil {
		return nil, err
	}

	dep.Spec.Template, err = patchTracing(params.Tempo, dep.Spec.Template)
	if err != nil {
		return nil, err
	}

	objs = append(objs, dep)
	return objs, nil
}

func httpScheme(tls bool) string {
	if tls {
		return "https"
	}
	return "http"
}

const containerNameTempoGateway = "tempo-gateway"

func resources(tempo v1alpha1.TempoStack) corev1.ResourceRequirements {
	if tempo.Spec.Template.Gateway.Resources == nil {
		return manifestutils.Resources(tempo, manifestutils.GatewayComponentName, nil)
	}
	return *tempo.Spec.Template.Gateway.Resources
}

// LivenessProbe returns the liveness probe spec for the gateway.
func LivenessProbe(tls bool) *corev1.Probe {
	scheme := corev1.URISchemeHTTP
	if tls {
		scheme = corev1.URISchemeHTTPS
	}

	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/live",
				Port:   intstr.FromString(manifestutils.GatewayInternalHttpPortName),
				Scheme: scheme,
			},
		},
		TimeoutSeconds:   2,
		PeriodSeconds:    30,
		FailureThreshold: 10,
	}
}

// ReadinessProbe returns the readiness probe for the gateway.
func ReadinessProbe(tls bool) *corev1.Probe {
	scheme := corev1.URISchemeHTTP
	if tls {
		scheme = corev1.URISchemeHTTPS
	}

	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/ready",
				Port:   intstr.FromString(manifestutils.GatewayInternalHttpPortName),
				Scheme: scheme,
			},
		},
		TimeoutSeconds:   1,
		PeriodSeconds:    5,
		FailureThreshold: 12,
	}
}

func deployment(params manifestutils.Params, rbacCfgHash string, tenantsCfgHash string) *appsv1.Deployment {
	tempo := params.Tempo
	labels := manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name)
	annotations := manifestutils.CommonAnnotations(params.ConfigChecksum)
	annotations["tempo.grafana.com/rbacConfig.hash"] = rbacCfgHash
	annotations["tempo.grafana.com/tenantsConfig.hash"] = tenantsCfgHash

	cfg := tempo.Spec.Template.Gateway
	tlsArgs := []string{}
	image := tempo.Spec.Images.TempoGateway
	if image == "" {
		image = params.CtrlConfig.DefaultImages.TempoGateway
	}

	if params.CtrlConfig.Gates.HTTPEncryption {
		tlsArgs = []string{
			fmt.Sprintf("--tls.internal.server.key-file=%s", path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSKeyFilename)),
			fmt.Sprintf("--tls.internal.server.cert-file=%s", path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSCertFilename)),
			fmt.Sprintf("--traces.tls.key-file=%s", path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSKeyFilename)),
			fmt.Sprintf("--traces.tls.cert-file=%s", path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSCertFilename)),
			fmt.Sprintf("--traces.tls.ca-file=%s", path.Join(manifestutils.TempoInternalTLSCADir, manifestutils.TLSCAFilename)),
		}
	}

	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.GatewayComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: tempo.Spec.ServiceAccount,
					NodeSelector:       cfg.NodeSelector,
					Tolerations:        cfg.Tolerations,
					Containers: []corev1.Container{
						{
							Name:  containerNameTempoGateway,
							Image: image,
							Env:   proxy.ReadProxyVarsFromEnv(),
							Args: append([]string{
								fmt.Sprintf("--traces.tenant-header=%s", manifestutils.TenantHeader),
								fmt.Sprintf("--web.listen=0.0.0.0:%d", manifestutils.GatewayPortHTTPServer),                                                                                             // proxies Tempo API and optionally Jaeger UI
								fmt.Sprintf("--web.internal.listen=0.0.0.0:%d", manifestutils.GatewayPortInternalHTTPServer),                                                                            // serves health checks
								fmt.Sprintf("--traces.write.endpoint=%s:%d", naming.ServiceFqdn(tempo.Namespace, tempo.Name, manifestutils.DistributorComponentName), manifestutils.PortOtlpGrpcServer), // Tempo Distributor gRPC upstream
								fmt.Sprintf("--traces.tempo.endpoint=%s://%s:%d", httpScheme(params.CtrlConfig.Gates.HTTPEncryption),
									naming.ServiceFqdn(tempo.Namespace, tempo.Name, manifestutils.QueryFrontendComponentName), manifestutils.PortHTTPServer), // Tempo API upstream
								fmt.Sprintf("--grpc.listen=0.0.0.0:%d", manifestutils.GatewayPortGRPCServer), // proxies Tempo Distributor gRPC
								fmt.Sprintf("--rbac.config=%s", path.Join(tempoGatewayMountDir, "cm", manifestutils.GatewayRBACFileName)),
								fmt.Sprintf("--tenants.config=%s", path.Join(tempoGatewayMountDir, "secret", manifestutils.GatewayTenantFileName)),
								"--log.level=info",
							}, tlsArgs...),
							Ports: []corev1.ContainerPort{
								{
									Name:          manifestutils.GatewayGrpcPortName,
									ContainerPort: manifestutils.GatewayPortGRPCServer,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          manifestutils.GatewayInternalHttpPortName,
									ContainerPort: manifestutils.GatewayPortInternalHTTPServer,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          manifestutils.GatewayHttpPortName,
									ContainerPort: manifestutils.GatewayPortHTTPServer,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							LivenessProbe:  LivenessProbe(params.CtrlConfig.Gates.HTTPEncryption),
							ReadinessProbe: ReadinessProbe(params.CtrlConfig.Gates.HTTPEncryption),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "rbac",
									ReadOnly:  true,
									MountPath: path.Join(tempoGatewayMountDir, "cm"),
								},
								{
									Name:      "tenant",
									ReadOnly:  true,
									MountPath: path.Join(tempoGatewayMountDir, "secret", manifestutils.GatewayTenantFileName),
									SubPath:   manifestutils.GatewayTenantFileName,
								},
							},
							Resources:       resources(tempo),
							SecurityContext: manifestutils.TempoContainerSecurityContext(),
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "rbac",
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
							Name: "tenant",
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
					},
				},
			},
		},
	}

	return dep
}

func patchTraceReadEndpoint(params manifestutils.Params, pod corev1.PodTemplateSpec) (corev1.PodTemplateSpec, error) {
	if !params.Tempo.Spec.Template.QueryFrontend.JaegerQuery.Enabled {
		return pod, nil
	}

	container := corev1.Container{
		Args: []string{
			fmt.Sprintf("--traces.read.endpoint=%s://%s:%d", httpScheme(params.CtrlConfig.Gates.HTTPEncryption),
				naming.ServiceFqdn(params.Tempo.Namespace, params.Tempo.Name, manifestutils.QueryFrontendComponentName), manifestutils.PortJaegerQuery), // Jaeger UI upstream
		},
	}

	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name != containerNameTempoGateway {
			continue
		}
		if err := mergo.Merge(&pod.Spec.Containers[i], container, mergo.WithAppendSlice); err != nil {
			return corev1.PodTemplateSpec{}, err
		}
	}

	return pod, nil
}

func patchTracing(tempo v1alpha1.TempoStack, pod corev1.PodTemplateSpec) (corev1.PodTemplateSpec, error) {
	if tempo.Spec.Observability.Tracing.SamplingFraction == "" {
		return pod, nil
	}

	host, port, err := net.SplitHostPort(tempo.Spec.Observability.Tracing.JaegerAgentEndpoint)
	if err != nil {
		return corev1.PodTemplateSpec{}, err
	}

	container := corev1.Container{
		Args: []string{
			fmt.Sprintf("--internal.tracing.endpoint=%s:%s", host, port),
			"--internal.tracing.endpoint-type=agent",
			fmt.Sprintf("--internal.tracing.sampling-fraction=%s", tempo.Spec.Observability.Tracing.SamplingFraction),
		},
	}

	for i := range pod.Spec.Containers {
		if err := mergo.Merge(&pod.Spec.Containers[i], container, mergo.WithAppendSlice); err != nil {
			return corev1.PodTemplateSpec{}, err
		}
	}

	return pod, mergo.Merge(&pod.Annotations, map[string]string{
		"sidecar.opentelemetry.io/inject": "true",
	})
}

func service(tempo v1alpha1.TempoStack, ocpServingCerts bool) *corev1.Service {
	labels := manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name)
	annotations := map[string]string{}
	if ocpServingCerts {
		annotations["service.beta.openshift.io/serving-cert-secret-name"] = naming.Name("gateway-tls", tempo.Name)
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Name(manifestutils.GatewayComponentName, tempo.Name),
			Namespace:   tempo.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       manifestutils.GatewayGrpcPortName,
					Port:       manifestutils.GatewayPortGRPCServer,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(manifestutils.GatewayGrpcPortName),
				},
				{
					Name:       manifestutils.GatewayInternalHttpPortName,
					Port:       manifestutils.GatewayPortInternalHTTPServer,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(manifestutils.GatewayInternalHttpPortName),
				},
				{
					Name:       manifestutils.GatewayHttpPortName,
					Port:       manifestutils.GatewayPortHTTPServer,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(manifestutils.GatewayHttpPortName),
				},
			},
			Selector: labels,
		},
	}
}

// NewRBACConfigMap creates a ConfigMap containing the RBAC configuration file.
func NewRBACConfigMap(cfgOpts options, namespace string, name string, labels labels.Set) (*corev1.ConfigMap, string, error) {
	rbacCfg, err := buildRBACConfig(cfgOpts)
	if err != nil {
		return nil, "", err
	}

	config := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    labels,
			Namespace: namespace,
		},
		Data: map[string]string{
			manifestutils.GatewayRBACFileName: rbacCfg,
		},
	}

	h := sha256.New()
	h.Write([]byte(rbacCfg))
	checksum := hex.EncodeToString(h.Sum(nil))

	return &config, checksum, nil
}

// NewTenantsSecret creates a Secret containing the tenants configuration file.
func NewTenantsSecret(cfgOpts options, namespace string, name string, labels labels.Set) (*corev1.Secret, string, error) {
	tenantsCfg, err := buildTenantsConfig(cfgOpts)
	if err != nil {
		return nil, "", err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    labels,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			manifestutils.GatewayTenantFileName: []byte(tenantsCfg),
		},
	}

	h := sha256.New()
	h.Write([]byte(tenantsCfg))
	checksum := hex.EncodeToString(h.Sum(nil))

	return secret, checksum, nil
}

func ingress(tempo v1alpha1.TempoStack) *networkingv1.Ingress {
	ingressName := naming.Name(manifestutils.GatewayComponentName, tempo.Name)
	labels := manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name)

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ingressName,
			Namespace:   tempo.Namespace,
			Labels:      labels,
			Annotations: tempo.Spec.Template.Gateway.Ingress.Annotations,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: tempo.Spec.Template.Gateway.Ingress.IngressClassName,
		},
	}

	backend := networkingv1.IngressBackend{
		Service: &networkingv1.IngressServiceBackend{
			Name: ingressName,
			Port: networkingv1.ServiceBackendPort{
				Name: "public",
			},
		},
	}

	if tempo.Spec.Template.Gateway.Ingress.Host == "" {
		ingress.Spec.DefaultBackend = &backend
	} else {
		pathType := networkingv1.PathTypePrefix
		ingress.Spec.Rules = []networkingv1.IngressRule{
			{
				Host: tempo.Spec.Template.Gateway.Ingress.Host,
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

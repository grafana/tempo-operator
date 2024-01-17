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
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

const (
	// tempoGatewayRbacFileName is the name of the rbac config file in the configmap.
	tempoGatewayRbacFileName = "rbac.yaml"

	// tempoGatewayMountDir is the path that is mounted from the configmap.
	tempoGatewayMountDir = "/etc/tempo-gateway"

	portGRPC = 8090

	// InternalPortName is the name of the gateway's internal port.
	InternalPortName = "internal"
	portInternal     = 8081

	portPublic = 8080
)

// BuildGateway creates gateway objects.
func BuildGateway(params manifestutils.Params) ([]client.Object, error) {
	rbacCfg, tenantsCfg, err := buildConfigFiles(newOptions(params.Tempo, params.CtrlConfig.Gates.OpenShift.BaseDomain, params.GatewayTenantSecret, params.GatewayTenantsData))
	if err != nil {
		return nil, err
	}

	rbacConfigMap, rbacCfgHash := rbacConfig(params.Tempo, rbacCfg)
	tenantsSecret, tenantsCfgHash := tenantsConfig(params.Tempo, tenantsCfg)

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
			clusterRole(params.Tempo),
			clusterRoleBinding(params.Tempo),
			serviceAccount(params.Tempo),
			configMapCABundle(params.Tempo),
		}...)

		if params.CtrlConfig.Gates.OpenShift.ServingCertsService {
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

func deployment(params manifestutils.Params, rbacCfgHash string, tenantsCfgHash string) *appsv1.Deployment {
	tempo := params.Tempo
	labels := manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name)
	annotations := manifestutils.CommonAnnotations(params.ConfigChecksum)
	annotations["tempo.grafana.com/rbacConfig.hash"] = rbacCfgHash
	annotations["tempo.grafana.com/tenantsConfig.hash"] = tenantsCfgHash

	cfg := tempo.Spec.Template.Gateway
	internalServerScheme := corev1.URISchemeHTTP
	tlsArgs := []string{}
	image := tempo.Spec.Images.TempoGateway
	if image == "" {
		image = params.CtrlConfig.DefaultImages.TempoGateway
	}

	if params.CtrlConfig.Gates.HTTPEncryption {
		internalServerScheme = corev1.URISchemeHTTPS
		tlsArgs = []string{
			fmt.Sprintf("--tls.internal.server.key-file=%s/tls.key", manifestutils.TempoServerTLSDir()),
			fmt.Sprintf("--tls.internal.server.cert-file=%s/tls.crt", manifestutils.TempoServerTLSDir()),
			fmt.Sprintf("--traces.tls.key-file=%s/tls.key", manifestutils.TempoServerTLSDir()),
			fmt.Sprintf("--traces.tls.cert-file=%s/tls.crt", manifestutils.TempoServerTLSDir()),
			fmt.Sprintf("--traces.tls.ca-file=%s/service-ca.crt", manifestutils.CABundleDir),
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
								fmt.Sprintf("--web.listen=0.0.0.0:%d", portPublic),
								fmt.Sprintf("--web.internal.listen=0.0.0.0:%d", portInternal),
								fmt.Sprintf("--traces.write.endpoint=%s:%d", naming.ServiceFqdn(tempo.Namespace, tempo.Name, manifestutils.DistributorComponentName), manifestutils.PortOtlpGrpcServer),
								fmt.Sprintf("--traces.tempo.endpoint=%s://%s:%d", httpScheme(params.CtrlConfig.Gates.HTTPEncryption),
									naming.ServiceFqdn(tempo.Namespace, tempo.Name, manifestutils.QueryFrontendComponentName), manifestutils.PortHTTPServer),
								fmt.Sprintf("--grpc.listen=0.0.0.0:%d", portGRPC),
								fmt.Sprintf("--rbac.config=%s", path.Join(tempoGatewayMountDir, "cm", tempoGatewayRbacFileName)),
								fmt.Sprintf("--tenants.config=%s", path.Join(tempoGatewayMountDir, "secret", manifestutils.GatewayTenantFileName)),
								"--log.level=info",
							}, tlsArgs...),
							Ports: []corev1.ContainerPort{
								{
									Name:          "grpc-public",
									ContainerPort: portGRPC,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "internal",
									ContainerPort: portInternal,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "public",
									ContainerPort: portPublic,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/live",
										Port:   intstr.FromInt(portInternal),
										Scheme: internalServerScheme,
									},
								},
								TimeoutSeconds:   2,
								PeriodSeconds:    30,
								FailureThreshold: 10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/ready",
										Port:   intstr.FromInt(portInternal),
										Scheme: internalServerScheme,
									},
								},
								TimeoutSeconds:   1,
								PeriodSeconds:    5,
								FailureThreshold: 12,
							},
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
							Resources:       manifestutils.Resources(tempo, manifestutils.GatewayComponentName, nil),
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
											Key:  tempoGatewayRbacFileName,
											Path: tempoGatewayRbacFileName,
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
				naming.ServiceFqdn(params.Tempo.Namespace, params.Tempo.Name, manifestutils.QueryFrontendComponentName), manifestutils.PortJaegerQuery),
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
					Name:       "grpc-public",
					Port:       portGRPC,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(portGRPC),
				},
				{
					Name:       InternalPortName,
					Port:       portInternal,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(portInternal),
				},
				{
					Name:       "public",
					Port:       portPublic,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(portPublic),
				},
			},
			Selector: labels,
		},
	}
}

func rbacConfig(tempo v1alpha1.TempoStack, rbacCfg string) (*corev1.ConfigMap, string) {
	labels := manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name)

	config := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.GatewayComponentName, tempo.Name),
			Labels:    labels,
			Namespace: tempo.Namespace,
		},
		Data: map[string]string{
			tempoGatewayRbacFileName: rbacCfg,
		},
	}

	h := sha256.New()
	h.Write([]byte(rbacCfg))
	checksum := hex.EncodeToString(h.Sum(nil))

	return &config, checksum
}

func tenantsConfig(tempo v1alpha1.TempoStack, tenantsCfg string) (*corev1.Secret, string) {
	labels := manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.GatewayComponentName, tempo.Name),
			Labels:    labels,
			Namespace: tempo.Namespace,
		},
		Data: map[string][]byte{
			manifestutils.GatewayTenantFileName: []byte(tenantsCfg),
		},
	}

	h := sha256.New()
	h.Write([]byte(tenantsCfg))
	checksum := hex.EncodeToString(h.Sum(nil))

	return secret, checksum
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

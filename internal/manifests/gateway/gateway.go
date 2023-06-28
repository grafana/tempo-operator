package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"path"

	"github.com/imdario/mergo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
	rbacCfg, tenantsCfg, err := buildConfigFiles(newOptions(params.Tempo, params.Gates.OpenShift.BaseDomain, params.GatewayTenantSecret, params.GatewayTenantsData))
	if err != nil {
		return nil, err
	}

	rbacConfigMap, rbacCfgHash := rbacConfig(params.Tempo, rbacCfg)
	tenantsSecret, tenantsCfgHash := tenantsConfig(params.Tempo, tenantsCfg)

	objs := []client.Object{
		rbacConfigMap,
		tenantsSecret,
		service(params.Tempo, params.Gates.OpenShift.ServingCertsService),
	}

	dep := deployment(params, rbacCfgHash, tenantsCfgHash)

	if params.Gates.HTTPEncryption || params.Gates.GRPCEncryption {
		caBundleName := naming.SigningCABundleName(params.Tempo.Name)
		if err := manifestutils.ConfigureServiceCA(&dep.Spec.Template.Spec, caBundleName); err != nil {
			return nil, err
		}
		err := manifestutils.ConfigureServicePKI(params.Tempo.Name, manifestutils.GatewayComponentName, &dep.Spec.Template.Spec)
		if err != nil {
			return nil, err
		}
	}

	if params.Tempo.Spec.Tenants.Mode == v1alpha1.OpenShift {
		dep = patchOCPServiceAccount(params.Tempo, dep)
		dep, err = patchOCPOPAContainer(params.Tempo, dep)
		if err != nil {
			return nil, err
		}

		objs = append(objs, []client.Object{
			clusterRole(params.Tempo),
			clusterRoleBinding(params.Tempo),
			serviceAccount(params.Tempo),
			configMapCABundle(params.Tempo),
		}...)

		if params.Gates.OpenShift.ServingCertsService {
			dep, err = patchOCPServingCerts(params.Tempo, dep)
			if err != nil {
				return nil, err
			}
		}
		if params.Gates.OpenShift.OpenShiftRoute {
			objs = append(objs, route(params.Tempo))
		}
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

func deployment(params manifestutils.Params, rbacCfgHash string, tenantsCfgHash string) *appsv1.Deployment {
	tempo := params.Tempo
	labels := manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name)
	annotations := manifestutils.CommonAnnotations(params.ConfigChecksum)
	annotations["tempo.grafana.com/rbacConfig.hash"] = rbacCfgHash
	annotations["tempo.grafana.com/tenantsConfig.hash"] = tenantsCfgHash

	cfg := tempo.Spec.Template.Gateway

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
							Name:  "tempo-gateway",
							Image: tempo.Spec.Images.TempoGateway,
							Args: []string{
								fmt.Sprintf("--traces.tenant-header=%s", manifestutils.TenantHeader),
								fmt.Sprintf("--web.listen=0.0.0.0:%d", portPublic),
								fmt.Sprintf("--web.internal.listen=0.0.0.0:%d", portInternal),
								fmt.Sprintf("--traces.write.endpoint=%s:4317", naming.ServiceFqdn(tempo.Namespace, tempo.Name, manifestutils.DistributorComponentName)),
								fmt.Sprintf("--traces.read.endpoint=%s://%s:16686", httpScheme(params.Gates.HTTPEncryption),
									naming.ServiceFqdn(tempo.Namespace, tempo.Name, manifestutils.QueryFrontendComponentName)),
								fmt.Sprintf("--grpc.listen=0.0.0.0:%d", portGRPC),
								fmt.Sprintf("--rbac.config=%s", path.Join(tempoGatewayMountDir, "cm", tempoGatewayRbacFileName)),
								fmt.Sprintf("--tenants.config=%s", path.Join(tempoGatewayMountDir, "secret", manifestutils.GatewayTenantFileName)),
								"--log.level=info",
							},
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
										Scheme: corev1.URISchemeHTTP,
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
										Scheme: corev1.URISchemeHTTP,
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
							Resources:       manifestutils.Resources(tempo, manifestutils.GatewayComponentName),
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

	if params.Gates.HTTPEncryption {
		dep.Spec.Template.Spec.Containers[0].Args = append(dep.Spec.Template.Spec.Containers[0].Args,
			fmt.Sprintf("--traces.tls.key-file=%s/tls.key", manifestutils.TempoServerTLSDir()),
			fmt.Sprintf("--traces.tls.cert-file=%s/tls.crt", manifestutils.TempoServerTLSDir()),
			fmt.Sprintf("--traces.tls.ca-file=%s/service-ca.crt", manifestutils.CABundleDir),
		)
	}

	return dep
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

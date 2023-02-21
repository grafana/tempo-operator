package gateway

import (
	"bytes"
	"embed"
	"fmt"
	"path"
	"text/template"

	"github.com/imdario/mergo"
	routev1 "github.com/openshift/api/route/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tempov1alpha1 "github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	"github.com/os-observability/tempo-operator/internal/manifests/naming"
)

const (
	// tempoComponentName is the name of the build tempo component.
	tempoComponentName = "gateway"
	// tempoGatewayTenantFileName is the name of the tenant config file in the configmap.
	tempoGatewayTenantFileName = "tenants.yaml"
	// tempoGatewayRbacFileName is the name of the rbac config file in the configmap.
	tempoGatewayRbacFileName = "rbac.yaml"

	// tempoGatewayMountDir is the path that is mounted from the configmap.
	tempoGatewayMountDir = "/etc/tempo-gateway"
)

var (
	//go:embed gateway-rbac.yaml
	tempoGatewayRbacYAMLTmplFile embed.FS

	//go:embed gateway-tenants.yaml
	tempoGatewayTenantsYAMLTmplFile embed.FS

	rbacTemplate = template.Must(template.ParseFS(tempoGatewayRbacYAMLTmplFile, "gateway-rbac.yaml"))

	tenantsTemplate = template.Must(template.ParseFS(tempoGatewayTenantsYAMLTmplFile, "gateway-tenants.yaml"))
)

// BuildGateway creates gateway objects.
func BuildGateway(params manifestutils.Params) ([]client.Object, error) {
	if !params.Tempo.Spec.Components.Gateway.Enabled ||
		params.Tempo.Spec.Tenants == nil {
		return []client.Object{}, nil
	}
	rbacCfg, tenantsCfg, err := getCfgs(options{
		Namespace: params.Tempo.Namespace,
		Name:      params.Tempo.Name,
		Tenants:   params.Tempo.Spec.Tenants,
	})
	if err != nil {
		return nil, err
	}

	objs := []client.Object{
		configMap(params.Tempo, rbacCfg),
		secrert(params.Tempo, tenantsCfg),
		service(params.Tempo, params.Gates.OpenShift.ServingCertsService),
		clusterRole(params.Tempo),
		clusterRoleBinding(params.Tempo),
		// TODO create conditionally https://github.com/os-observability/tempo-operator/issues/243
		serviceAccount(params.Tempo),
	}

	if params.Gates.OpenShift.GatewayRoute {
		objs = append(objs, route(params.Tempo))
	}
	dep := deployment(params)
	if params.Gates.OpenShift.ServingCertsService {
		dep, err = patchOCPServingCerts(params.Tempo, dep)
		if err != nil {
			return nil, err
		}
	}

	objs = append(objs, dep)
	return objs, nil
}

// generate gateway configuration files.
func getCfgs(opts options) (rbacCfg string, tenantsCfg string, err error) {
	// Build tempo gateway rbac yaml
	byteBuffer := &bytes.Buffer{}
	err = rbacTemplate.Execute(byteBuffer, opts)
	if err != nil {
		return "", "", fmt.Errorf("failed to create tempo gateway rbac configuration, err: %w", err)
	}
	rbacCfg = byteBuffer.String()
	// Build tempo gateway tenants yaml
	byteBuffer.Reset()
	err = tenantsTemplate.Execute(byteBuffer, opts)
	if err != nil {
		return "", "", fmt.Errorf("failed to create tempo gateway tenants configuration, err: %w", err)
	}
	tenantsCfg = byteBuffer.String()
	return rbacCfg, tenantsCfg, nil
}

func serviceAccount(tempo tempov1alpha1.Microservices) *corev1.ServiceAccount {
	tt := true
	labels := manifestutils.ComponentLabels(tempoComponentName, tempo.Name)
	annotations := map[string]string{}
	for _, tenantAuth := range tempo.Spec.Tenants.Authentication {
		key := fmt.Sprintf("serviceaccounts.openshift.io/oauth-redirectreference.%s", tenantAuth.TenantName)
		val := fmt.Sprintf(`{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"%s"}}`, tempo.Name)
		annotations[key] = val
	}
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Name(tempoComponentName, tempo.Name),
			Namespace:   tempo.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		AutomountServiceAccountToken: &tt,
	}
}

func clusterRole(tempo tempov1alpha1.Microservices) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(tempoComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    manifestutils.ComponentLabels(tempoComponentName, tempo.Name),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"authentication.k8s.io"},
				Resources: []string{"tokenreviews"},
				Verbs:     []string{"create"},
			},
		},
	}
}

func clusterRoleBinding(tempo tempov1alpha1.Microservices) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(tempoComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    manifestutils.ComponentLabels(tempoComponentName, tempo.Name),
		},
		Subjects: []rbacv1.Subject{
			{
				Name:      naming.Name(tempoComponentName, tempo.Name),
				Kind:      "ServiceAccount",
				Namespace: tempo.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     naming.Name(tempoComponentName, tempo.Name),
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

const (
	portGRPC     = 8090
	portInternal = 8081
	portPublic   = 8080
)

func deployment(params manifestutils.Params) *v1.Deployment {
	tempo := params.Tempo
	labels := manifestutils.ComponentLabels(tempoComponentName, tempo.Name)
	annotations := manifestutils.CommonAnnotations(params.ConfigChecksum)
	cfg := tempo.Spec.Components.Gateway

	return &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(tempoComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					// Gateway needs elevated permissions to call token review API
					ServiceAccountName: naming.Name(tempoComponentName, tempo.Name),
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
								fmt.Sprintf("--traces.write.endpoint=%s:4317", naming.Name(manifestutils.DistributorComponentName, tempo.Name)),
								fmt.Sprintf("--traces.read.endpoint=http://%s:16686", naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name)),
								fmt.Sprintf("--grpc.listen=0.0.0.0:%d", portGRPC),
								fmt.Sprintf("--rbac.config=%s", path.Join(tempoGatewayMountDir, "cm", tempoGatewayRbacFileName)),
								fmt.Sprintf("--tenants.config=%s", path.Join(tempoGatewayMountDir, "secert", tempoGatewayTenantFileName)),
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
							// TODO enable once probles are not secured by TLS https://github.com/os-observability/tempo-operator/issues/241
							//LivenessProbe: &corev1.Probe{
							//	ProbeHandler: corev1.ProbeHandler{
							//		HTTPGet: &corev1.HTTPGetAction{
							//			Path:   "/live",
							//			Port:   intstr.FromInt(portInternal),
							//			Scheme: corev1.URISchemeHTTP,
							//		},
							//	},
							//	TimeoutSeconds:   2,
							//	PeriodSeconds:    30,
							//	FailureThreshold: 10,
							//},
							//ReadinessProbe: &corev1.Probe{
							//	ProbeHandler: corev1.ProbeHandler{
							//		HTTPGet: &corev1.HTTPGetAction{
							//			Path:   "/ready",
							//			Port:   intstr.FromInt(portInternal),
							//			Scheme: corev1.URISchemeHTTP,
							//		},
							//	},
							//	TimeoutSeconds:   1,
							//	PeriodSeconds:    5,
							//	FailureThreshold: 12,
							//},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "rbac",
									ReadOnly:  true,
									MountPath: path.Join(tempoGatewayMountDir, "cm"),
								},
								{
									Name:      "tenant",
									ReadOnly:  true,
									MountPath: path.Join(tempoGatewayMountDir, "secert", tempoGatewayTenantFileName),
									SubPath:   tempoGatewayTenantFileName,
								},
							},
							// TODO(frzifus): add gateway to resource pool.
							// Resources:       manifestutils.Resources(tempo, tempoComponentName),
							SecurityContext: manifestutils.TempoContainerSecurityContext(),
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "rbac",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: naming.Name(tempoComponentName, tempo.Name),
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
									SecretName: naming.Name(tempoComponentName, tempo.Name),
									Items: []corev1.KeyToPath{
										{
											Key:  tempoGatewayTenantFileName,
											Path: tempoGatewayTenantFileName,
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
}

func patchOCPServingCerts(tempo tempov1alpha1.Microservices, dep *v1.Deployment) (*v1.Deployment, error) {
	container := corev1.Container{
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "serving-certs",
				ReadOnly:  true,
				MountPath: path.Join(tempoGatewayMountDir, "serving-certs"),
			},
		},
		Args: []string{
			fmt.Sprintf("--tls.server.cert-file=%s", path.Join(tempoGatewayMountDir, "serving-certs", "tls.crt")),
			fmt.Sprintf("--tls.server.key-file=%s", path.Join(tempoGatewayMountDir, "serving-certs", "tls.key")),
		},
	}
	err := mergo.Merge(&dep.Spec.Template.Spec.Containers[0], container, mergo.WithAppendSlice)
	if err != nil {
		return nil, err
	}

	pod := corev1.PodSpec{
		Volumes: []corev1.Volume{
			{
				Name: "serving-certs",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: naming.Name("gateway-tls", tempo.Name),
					},
				},
			},
		},
	}
	err = mergo.Merge(&dep.Spec.Template.Spec, pod, mergo.WithAppendSlice)
	if err != nil {
		return nil, err
	}
	return dep, err
}

func service(tempo tempov1alpha1.Microservices, ocpServingCerts bool) *corev1.Service {
	labels := manifestutils.ComponentLabels(tempoComponentName, tempo.Name)
	annotations := map[string]string{}
	if ocpServingCerts {
		annotations["service.beta.openshift.io/serving-cert-secret-name"] = naming.Name("gateway-tls", tempo.Name)
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Name(tempoComponentName, tempo.Name),
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
					Name:       "internal",
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

func configMap(tempo tempov1alpha1.Microservices, rbacCfg string) *corev1.ConfigMap {
	labels := manifestutils.ComponentLabels(tempoComponentName, tempo.Name)
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(tempoComponentName, tempo.Name),
			Labels:    labels,
			Namespace: tempo.Namespace,
		},
		Data: map[string]string{
			tempoGatewayRbacFileName: rbacCfg,
		},
	}
}

func route(tempo tempov1alpha1.Microservices) *routev1.Route {
	labels := manifestutils.ComponentLabels(tempoComponentName, tempo.Name)
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name("", tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: naming.Name(tempoComponentName, tempo.Name),
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.IntOrString{
					StrVal: "public",
				},
			},
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationPassthrough,
			},
			WildcardPolicy: routev1.WildcardPolicyNone,
		},
	}
}

func secrert(tempo tempov1alpha1.Microservices, tenantsCfg string) *corev1.Secret {
	labels := manifestutils.ComponentLabels(tempoComponentName, tempo.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(tempoComponentName, tempo.Name),
			Labels:    labels,
			Namespace: tempo.Namespace,
		},
		Data: map[string][]byte{
			tempoGatewayTenantFileName: []byte(tenantsCfg),
		},
	}
	return secret
}

// options is used to render the rbac.yaml and tenants.yaml file template.
type options struct {
	Namespace     string
	Name          string
	Tenants       *tempov1alpha1.TenantsSpec
	TenantSecrets []*secret
}

// secret for clientID, clientSecret and issuerCAPath for tenant's authentication.
type secret struct {
	TenantName   string
	ClientID     string
	ClientSecret string
	IssuerCAPath string
}

package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"

	"github.com/imdario/mergo"
	routev1 "github.com/openshift/api/route/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	"github.com/os-observability/tempo-operator/internal/manifests/naming"
)

const (
	// tempoGatewayTenantFileName is the name of the tenant config file in the configmap.
	tempoGatewayTenantFileName = "tenants.yaml"
	// tempoGatewayRbacFileName is the name of the rbac config file in the configmap.
	tempoGatewayRbacFileName = "rbac.yaml"

	// tempoGatewayMountDir is the path that is mounted from the configmap.
	tempoGatewayMountDir = "/etc/tempo-gateway"

	portGRPC     = 8090
	portInternal = 8081
	portPublic   = 8080
)

// BuildGateway creates gateway objects.
func BuildGateway(params manifestutils.Params) ([]client.Object, error) {
	if !params.Tempo.Spec.Template.Gateway.Enabled ||
		params.Tempo.Spec.Tenants == nil {
		return []client.Object{}, nil
	}

	rbacCfg, tenantsCfg, err := buildConfigFiles(options{
		Namespace:     params.Tempo.Namespace,
		Name:          params.Tempo.Name,
		BaseDomain:    params.Gates.OpenShift.BaseDomain,
		Tenants:       params.Tempo.Spec.Tenants,
		TenantSecrets: params.GatewayTenantSecret,
	})
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

	if params.Tempo.Spec.Tenants.Mode == v1alpha1.OpenShift {
		dep = patchServiceAccount(params.Tempo, dep)
		objs = append(objs, []client.Object{
			clusterRole(params.Tempo),
			clusterRoleBinding(params.Tempo),
			serviceAccount(params.Tempo),
		}...)
		if params.Gates.OpenShift.ServingCertsService {
			dep, err = patchOCPServingCerts(params.Tempo, dep)
			if err != nil {
				return nil, err
			}
		}
		if params.Gates.OpenShift.GatewayRoute {
			objs = append(objs, route(params.Tempo))
		}
	}

	objs = append(objs, dep)
	return objs, nil
}

func serviceAccount(tempo v1alpha1.TempoStack) *corev1.ServiceAccount {
	tt := true
	labels := manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name)
	annotations := map[string]string{}
	for _, tenantAuth := range tempo.Spec.Tenants.Authentication {
		key := fmt.Sprintf("serviceaccounts.openshift.io/oauth-redirectreference.%s", tenantAuth.TenantName)
		val := fmt.Sprintf(`{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"%s"}}`, naming.Name(manifestutils.GatewayComponentName, tempo.Name))
		annotations[key] = val
	}
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Name(manifestutils.GatewayComponentName, tempo.Name),
			Namespace:   tempo.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		AutomountServiceAccountToken: &tt,
	}
}

func clusterRole(tempo v1alpha1.TempoStack) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   naming.Name(manifestutils.GatewayComponentName, tempo.Name),
			Labels: manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name),
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

func clusterRoleBinding(tempo v1alpha1.TempoStack) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   naming.Name(manifestutils.GatewayComponentName, tempo.Name),
			Labels: manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name),
		},
		Subjects: []rbacv1.Subject{
			{
				Name:      naming.Name(manifestutils.GatewayComponentName, tempo.Name),
				Kind:      "ServiceAccount",
				Namespace: tempo.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     naming.Name(manifestutils.GatewayComponentName, tempo.Name),
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func deployment(params manifestutils.Params, rbacCfgHash string, tenantsCfgHash string) *v1.Deployment {
	tempo := params.Tempo
	labels := manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name)
	annotations := manifestutils.CommonAnnotations(params.ConfigChecksum)
	annotations["tempo.grafana.com/rbacConfig.hash"] = rbacCfgHash
	annotations["tempo.grafana.com/tenantsConfig.hash"] = tenantsCfgHash

	cfg := tempo.Spec.Template.Gateway

	return &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.GatewayComponentName, tempo.Name),
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
								fmt.Sprintf("--traces.write.endpoint=%s:4317", naming.Name(manifestutils.DistributorComponentName, tempo.Name)),
								fmt.Sprintf("--traces.read.endpoint=http://%s:16686", naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name)),
								fmt.Sprintf("--grpc.listen=0.0.0.0:%d", portGRPC),
								fmt.Sprintf("--rbac.config=%s", path.Join(tempoGatewayMountDir, "cm", tempoGatewayRbacFileName)),
								fmt.Sprintf("--tenants.config=%s", path.Join(tempoGatewayMountDir, "secret", tempoGatewayTenantFileName)),
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
									MountPath: path.Join(tempoGatewayMountDir, "secret", tempoGatewayTenantFileName),
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

func patchOCPServingCerts(tempo v1alpha1.TempoStack, dep *v1.Deployment) (*v1.Deployment, error) {
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

func patchServiceAccount(tempo v1alpha1.TempoStack, dep *v1.Deployment) *v1.Deployment {
	dep.Spec.Template.Spec.ServiceAccountName = naming.Name(manifestutils.GatewayComponentName, tempo.Name)
	return dep
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

func route(tempo v1alpha1.TempoStack) *routev1.Route {
	labels := manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name)
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.GatewayComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: naming.Name(manifestutils.GatewayComponentName, tempo.Name),
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("public"),
			},
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationPassthrough,
			},
			WildcardPolicy: routev1.WildcardPolicyNone,
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
			tempoGatewayTenantFileName: []byte(tenantsCfg),
		},
	}

	h := sha256.New()
	h.Write([]byte(tenantsCfg))
	checksum := hex.EncodeToString(h.Sum(nil))


	return secret, checksum
}

package gateway

import (
	"fmt"
	"path"

	"github.com/imdario/mergo"
	routev1 "github.com/openshift/api/route/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

const (
	gatewayOPAHTTPPort     = 8082
	gatewayOPAInternalPort = 8083

	timeoutRouteAnnotation = "haproxy.router.openshift.io/timeout"
)

// BuildServiceAccountAnnotations returns the annotations to use a ServiceAccount as an OAuth client.
func BuildServiceAccountAnnotations(tenants v1alpha1.TenantsSpec, routeName string) map[string]string {
	annotations := map[string]string{}
	for _, tenantAuth := range tenants.Authentication {
		key := fmt.Sprintf("serviceaccounts.openshift.io/oauth-redirectreference.%s", tenantAuth.TenantName)
		val := fmt.Sprintf(`{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"%s"}}`, routeName)
		annotations[key] = val
	}
	return annotations
}

func serviceAccount(tempo v1alpha1.TempoStack) *corev1.ServiceAccount {
	tt := true
	labels := manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name)
	annotations := BuildServiceAccountAnnotations(*tempo.Spec.Tenants, naming.Name(manifestutils.GatewayComponentName, tempo.Name))
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

// NewAccessReviewClusterRole creates a ClusterRole for tokenreviews.
func NewAccessReviewClusterRole(name string, labels labels.Set) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
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

// NewAccessReviewClusterRoleBinding creates a ClusterRoleBinding for the ClusterRole created by NewAccessReviewClusterRole().
func NewAccessReviewClusterRoleBinding(name string, labels labels.Set, saNamespace string, saName string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Subjects: []rbacv1.Subject{
			{
				Name:      saName,
				Kind:      "ServiceAccount",
				Namespace: saNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     name,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func route(tempo v1alpha1.TempoStack) (*routev1.Route, error) {
	labels := manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name)

	var tlsCfg *routev1.TLSConfig
	switch tempo.Spec.Template.Gateway.Ingress.Route.Termination {
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

	annotations := tempo.Spec.Template.Gateway.Ingress.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}
	if annotations[timeoutRouteAnnotation] == "" {
		annotations[timeoutRouteAnnotation] = fmt.Sprintf("%ds", int(tempo.Spec.Timeout.Duration.Seconds()))
	}

	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Name(manifestutils.GatewayComponentName, tempo.Name),
			Namespace:   tempo.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: routev1.RouteSpec{
			Host: tempo.Spec.Template.Gateway.Ingress.Host,
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: naming.Name(manifestutils.GatewayComponentName, tempo.Name),
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("public"),
			},
			TLS:            tlsCfg,
			WildcardPolicy: routev1.WildcardPolicyNone,
		},
	}, nil
}

func patchOCPServingCerts(tempo v1alpha1.TempoStack, dep *v1.Deployment) (*v1.Deployment, error) {
	container := corev1.Container{
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "serving-certs",
				ReadOnly:  true,
				MountPath: path.Join(tempoGatewayMountDir, "serving-certs"),
			},
			{
				Name:      "cabundle",
				ReadOnly:  true,
				MountPath: path.Join(tempoGatewayMountDir, "cabundle"),
			},
		},
		Args: []string{
			fmt.Sprintf("--tls.server.cert-file=%s", path.Join(tempoGatewayMountDir, "serving-certs", "tls.crt")),
			fmt.Sprintf("--tls.server.key-file=%s", path.Join(tempoGatewayMountDir, "serving-certs", "tls.key")),
			fmt.Sprintf("--tls.healthchecks.server-ca-file=%s", path.Join(tempoGatewayMountDir, "cabundle", "service-ca.crt")),
			fmt.Sprintf("--tls.healthchecks.server-name=%s", naming.ServiceFqdn(tempo.Namespace, tempo.Name, manifestutils.GatewayComponentName)),
			"--web.healthchecks.url=https://localhost:8080",
			"--tls.client-auth-type=NoClientCert",
		},
	}
	// WithOverrides overrides the HTTP in probes
	err := mergo.Merge(&dep.Spec.Template.Spec.Containers[0], container, mergo.WithAppendSlice, mergo.WithOverride)
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
			{
				Name: "cabundle",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: naming.Name("gateway-cabundle", tempo.Name),
						},
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

func patchOCPServiceAccount(tempo v1alpha1.TempoStack, dep *v1.Deployment) *v1.Deployment {
	dep.Spec.Template.Spec.ServiceAccountName = naming.Name(manifestutils.GatewayComponentName, tempo.Name)
	return dep
}

func patchOCPOPAContainer(params manifestutils.Params, dep *v1.Deployment) (*v1.Deployment, error) {
	pod := corev1.PodSpec{
		Containers: []corev1.Container{NewOpaContainer(params.CtrlConfig, *params.Tempo.Spec.Tenants, params.Tempo.Spec.Template.Gateway.RBAC.Enabled, "tempostack", corev1.ResourceRequirements{})},
	}
	err := mergo.Merge(&dep.Spec.Template.Spec, pod, mergo.WithAppendSlice)
	if err != nil {
		return nil, err
	}
	return dep, err
}

// NewOpaContainer creates an OPA (https://github.com/observatorium/opa-openshift) container.
func NewOpaContainer(ctrlConfig configv1alpha1.ProjectConfig, tenants v1alpha1.TenantsSpec, rbac bool, opaPackage string, resources corev1.ResourceRequirements) corev1.Container {
	var args = []string{
		"--log.level=warn",
		fmt.Sprintf("--web.listen=:%d", gatewayOPAHTTPPort),
		fmt.Sprintf("--web.internal.listen=:%d", gatewayOPAInternalPort),
		fmt.Sprintf("--web.healthchecks.url=http://localhost:%d", gatewayOPAHTTPPort),
		fmt.Sprintf("--opa.package=%s", opaPackage),
		"--opa.ssar",
	}
	if rbac {
		args = append(args, "--opa.matcher=kubernetes_namespace_name")
	}
	for _, t := range tenants.Authentication {
		args = append(args, fmt.Sprintf("--openshift.mappings=%s=%s", t.TenantName, "tempo.grafana.com"))
	}

	return corev1.Container{
		Name:  fmt.Sprintf("%s-opa", containerNameTempoGateway),
		Image: ctrlConfig.DefaultImages.TempoGatewayOpa,
		Args:  args,
		Ports: []corev1.ContainerPort{
			{
				Name:          "public",
				ContainerPort: gatewayOPAHTTPPort,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "opa-metrics",
				ContainerPort: gatewayOPAInternalPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/live",
					Port:   intstr.FromInt(gatewayOPAInternalPort),
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
					Port:   intstr.FromInt(gatewayOPAInternalPort),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			TimeoutSeconds:   1,
			PeriodSeconds:    5,
			FailureThreshold: 12,
		},
		Resources: resources,
	}
}

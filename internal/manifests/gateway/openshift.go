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
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

const (
	gatewayOPAHTTPPort     = 8082
	gatewayOPAInternalPort = 8083
)

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
			{
				APIGroups: []string{"authorization.k8s.io"},
				Resources: []string{"subjectaccessreviews"},
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

	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.GatewayComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
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

func configMapCABundle(tempo v1alpha1.TempoStack) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Name("gateway-cabundle", tempo.Name),
			Labels:      manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name),
			Annotations: map[string]string{"service.beta.openshift.io/inject-cabundle": "true"},
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

func patchOCPOPAContainer(tempo v1alpha1.TempoStack, dep *v1.Deployment) (*v1.Deployment, error) {
	pod := corev1.PodSpec{
		Containers: []corev1.Container{opaContainer(tempo)},
	}
	err := mergo.Merge(&dep.Spec.Template.Spec, pod, mergo.WithAppendSlice)
	if err != nil {
		return nil, err
	}
	return dep, err
}

func opaContainer(tempo v1alpha1.TempoStack) corev1.Container {
	var args = []string{
		"--log.level=warn",
		"--opa.admin-groups=system:cluster-admins,cluster-admin,dedicated-admin",
		fmt.Sprintf("--web.listen=:%d", gatewayOPAHTTPPort),
		fmt.Sprintf("--web.internal.listen=:%d", gatewayOPAInternalPort),
		fmt.Sprintf("--web.healthchecks.url=http://localhost:%d", gatewayOPAHTTPPort),
		fmt.Sprintf("--opa.package=%s", "tempostack"),
	}
	for _, t := range tempo.Spec.Tenants.Authentication {
		args = append(args, fmt.Sprintf(`--openshift.mappings=%s=%s`, t.TenantName, "tempo.grafana.com"))
	}

	return corev1.Container{
		Name:  "opa",
		Image: tempo.Spec.Images.TempoGatewayOpa,
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
	}
}

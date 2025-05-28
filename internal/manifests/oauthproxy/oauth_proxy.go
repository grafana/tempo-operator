package oauthproxy

import (
	"fmt"
	"strings"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/operator-framework/operator-lib/proxy"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

const (
	tlsProxyPath                           = "/etc/tls/private"
	healthPath                             = "/oauth/healthz"
	sessionSecretKey                       = "session_secret"
	oauthProxySecretMountPath              = "/etc/proxy/cookie/"
	oauthReadinessProbeInitialDelaySeconds = 10
	oauthReadinessProbeTimeoutSeconds      = 5
	serviceAccountRedirectAnnotation       = "serviceaccounts.openshift.io/oauth-redirectreference.primary"
	minBytesRequiredByCookieValue          = 16
)

// OAuthServiceAccount returns a service account representing a client in the context of the OAuth Proxy.
func OAuthServiceAccount(params manifestutils.Params) *corev1.ServiceAccount {
	labels := manifestutils.ComponentLabels(manifestutils.QueryFrontendComponentName, params.Tempo.Name)
	annotations := map[string]string{
		serviceAccountRedirectAnnotation: getOAuthRedirectReference(naming.Name(manifestutils.QueryFrontendComponentName, params.Tempo.Name)),
	}
	if params.StorageParams.S3 != nil && params.StorageParams.CredentialMode == v1alpha1.CredentialModeToken {
		awsAnnotations := manifestutils.S3AWSSTSAnnotations(*params.StorageParams.S3)
		for k, v := range awsAnnotations {
			annotations[k] = v
		}
	}

	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Name(manifestutils.QueryFrontendComponentName, params.Tempo.Name),
			Namespace:   params.Tempo.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
	}
}

// AddServiceAccountAnnotations add the redirect reference annotation to a service account, used by oauth proxy.
func AddServiceAccountAnnotations(serviceAccount *corev1.ServiceAccount, routeName string) {
	if serviceAccount.Annotations == nil {
		serviceAccount.Annotations = make(map[string]string)
	}
	serviceAccount.Annotations[serviceAccountRedirectAnnotation] = getOAuthRedirectReference(routeName)
}

// PatchRouteForOauthProxy a modified route pointing to the oauth proxy and annotated.
func PatchRouteForOauthProxy(route *routev1.Route) { // point route to the oauth proxy
	route.Spec.TLS = &routev1.TLSConfig{Termination: routev1.TLSTerminationReencrypt}
	route.Spec.Port.TargetPort = intstr.FromString(manifestutils.OAuthProxyPortName)
}

// PatchStatefulSetForOauthProxy returns a modified StatefulSet with the oauth sidecar container and the right service account.
func PatchStatefulSetForOauthProxy(
	tempo metav1.ObjectMeta,
	authSpec *v1alpha1.JaegerQueryAuthenticationSpec,
	timeout time.Duration,
	config configv1alpha1.ProjectConfig,
	statefulSet *v1.StatefulSet,
	defaultResources *corev1.ResourceRequirements,
) {
	statefulSet.Spec.Template.Spec.Volumes = append(statefulSet.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: getTLSSecretNameForFrontendService(tempo.Name),
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: getTLSSecretNameForFrontendService(tempo.Name),
			},
		},
	})

	statefulSet.Spec.Template.Spec.Containers = append(statefulSet.Spec.Template.Spec.Containers,
		oAuthProxyContainer(tempo.Name, statefulSet.Spec.Template.Spec.ServiceAccountName, authSpec, timeout,
			config.DefaultImages.OauthProxy, defaultResources))
}

// PatchDeploymentForOauthProxy returns a modified deployment with the oauth sidecar container and the right service account.
func PatchDeploymentForOauthProxy(
	tempo metav1.ObjectMeta,
	config configv1alpha1.ProjectConfig,
	authSpec *v1alpha1.JaegerQueryAuthenticationSpec,
	timeout time.Duration,
	imageSpec configv1alpha1.ImagesSpec,
	dep *v1.Deployment,
	defaultResources *corev1.ResourceRequirements,
) {
	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: getTLSSecretNameForFrontendService(tempo.Name),
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: getTLSSecretNameForFrontendService(tempo.Name),
			},
		},
	})

	dep.Spec.Template.Spec.ServiceAccountName = naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name)

	oauthProxyImage := imageSpec.OauthProxy
	if oauthProxyImage == "" {
		oauthProxyImage = config.DefaultImages.OauthProxy
	}

	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers,
		oAuthProxyContainer(tempo.Name,
			naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name),
			authSpec,
			timeout,
			oauthProxyImage,
			defaultResources,
		))
}

func getTLSSecretNameForFrontendService(tempoName string) string {
	return fmt.Sprintf("%s-ui-oauth-proxy-tls", tempoName)
}

func proxyInitArguments(serviceAccountName string, timeout time.Duration) []string {
	return []string{
		// The SA Token is injected by admission controller by adding a volume via pod mutation
		// In Kubernetes 1.24 the SA token is short-lived (default 1h)
		// The proxy loads the token at startup and uses it as secret to encrypt cookies.
		// The proxy does not reload the token.
		// If the token changes during lifetime of the proxy the already provisioned cookies
		// are not invalidated.
		// The SA token is invalidated when pod is deleted (or restarted) which logs out all users.
		// An alternative approach would be to randomly generate the secret in the reconciliation
		// loop and inject it as file/secret to directly via flag. The reconciliation loop would invalidate
		// The token on every run.
		"--cookie-secret-file=/var/run/secrets/kubernetes.io/serviceaccount/token",
		fmt.Sprintf("--https-address=:%d", manifestutils.OAuthProxyPort),
		fmt.Sprintf("--openshift-service-account=%s", serviceAccountName),
		"--provider=openshift",
		fmt.Sprintf("--tls-cert=%s/tls.crt", tlsProxyPath),
		fmt.Sprintf("--tls-key=%s/tls.key", tlsProxyPath),
		fmt.Sprintf("--upstream=http://localhost:%d", manifestutils.PortJaegerUI),
		fmt.Sprintf("--upstream-timeout=%s", timeout.String()),
	}
}

func oAuthProxyContainer(
	tempo string,
	serviceAccountName string,
	authSpec *v1alpha1.JaegerQueryAuthenticationSpec,
	timeout time.Duration,
	oauthProxyImage string,
	defaultResources *corev1.ResourceRequirements,
) corev1.Container {
	args := proxyInitArguments(serviceAccountName, timeout)

	if len(strings.TrimSpace(authSpec.SAR)) > 0 {
		args = append(args, fmt.Sprintf("--openshift-sar=%s", authSpec.SAR))
	}
	resources := authSpec.Resources
	if resources == nil {
		resources = defaultResources
	}

	return corev1.Container{
		Image: oauthProxyImage,
		Name:  "oauth-proxy",
		Args:  args,
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: manifestutils.OAuthProxyPort,
				Name:          manifestutils.OAuthProxyPortName,
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: tlsProxyPath,
				Name:      getTLSSecretNameForFrontendService(tempo),
			},
		},
		Resources: *resources,
		Env:       proxy.ReadProxyVarsFromEnv(),
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Scheme: corev1.URISchemeHTTPS,
					Path:   healthPath,
					Port:   intstr.FromString(manifestutils.OAuthProxyPortName),
				},
			},
			InitialDelaySeconds: oauthReadinessProbeInitialDelaySeconds,
			TimeoutSeconds:      oauthReadinessProbeTimeoutSeconds,
		},
		SecurityContext: manifestutils.TempoContainerSecurityContext(),
	}
}

// PatchQueryFrontEndService add necessary ports and annotations to the front end service.
func PatchQueryFrontEndService(service *corev1.Service, tempo string) {
	if service == nil {
		return
	}

	if service.Annotations == nil {
		service.Annotations = make(map[string]string)
	}

	service.Annotations["service.beta.openshift.io/serving-cert-secret-name"] = getTLSSecretNameForFrontendService(tempo)

	service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
		Name:       manifestutils.OAuthProxyPortName,
		Port:       manifestutils.OAuthProxyPort,
		TargetPort: intstr.FromString(manifestutils.OAuthProxyPortName),
	})
}

func getOAuthRedirectReference(routeName string) string {
	return fmt.Sprintf(
		`{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"%s"}}`,
		routeName)
}

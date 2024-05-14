package oauthproxy

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/operator-framework/operator-lib/proxy"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
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
func OAuthServiceAccount(tempo metav1.ObjectMeta) *corev1.ServiceAccount {
	labels := manifestutils.ComponentLabels(manifestutils.QueryFrontendComponentName, tempo.Name)
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				serviceAccountRedirectAnnotation: getOAuthRedirectReference(naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name)),
			},
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

// OAuthCookieSessionSecret returns a secret that contains the cookie secret used by oauth proxy.
func OAuthCookieSessionSecret(tempo metav1.ObjectMeta) (*corev1.Secret, error) {
	sessionSecret, err := generateProxySecret()

	if err != nil {
		return nil, err
	}

	labels := manifestutils.ComponentLabels(manifestutils.QueryFrontendOauthProxyComponentName, tempo.Name)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cookieSecretName(tempo.Name),
			Labels:    labels,
			Namespace: tempo.Namespace,
		},
		Data: map[string][]byte{
			sessionSecretKey: []byte(sessionSecret),
		},
	}, nil
}

// PatchStatefulSetForOauthProxy returns a modified StatefulSet with the oauth sidecar container and the right service account.
func PatchStatefulSetForOauthProxy(tempo metav1.ObjectMeta,
	authSpec *v1alpha1.JaegerQueryAuthenticationSpec,
	config configv1alpha1.ProjectConfig, statefulSet *v1.StatefulSet) {
	statefulSet.Spec.Template.Spec.Volumes = append(statefulSet.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: getTLSSecretNameForFrontendService(tempo.Name),
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: getTLSSecretNameForFrontendService(tempo.Name),
			},
		},
	})

	statefulSet.Spec.Template.Spec.Volumes = append(statefulSet.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: cookieSecretName(tempo.Name),
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: cookieSecretName(tempo.Name),
			},
		},
	})

	statefulSet.Spec.Template.Spec.Containers = append(statefulSet.Spec.Template.Spec.Containers,
		oAuthProxyContainer(tempo.Name, statefulSet.Spec.Template.Spec.ServiceAccountName, authSpec, config.DefaultImages.OauthProxy))
}

// PatchDeploymentForOauthProxy returns a modified deployment with the oauth sidecar container and the right service account.
func PatchDeploymentForOauthProxy(
	tempo metav1.ObjectMeta,
	config configv1alpha1.ProjectConfig,
	authSpec *v1alpha1.JaegerQueryAuthenticationSpec,
	imageSpec configv1alpha1.ImagesSpec,
	dep *v1.Deployment) {
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
		oAuthProxyContainer(tempo.Name, naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name),
			authSpec, oauthProxyImage))

	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: cookieSecretName(tempo.Name),
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: cookieSecretName(tempo.Name),
			},
		},
	})
}

func getTLSSecretNameForFrontendService(tempoName string) string {
	return fmt.Sprintf("%s-ui-oauth-proxy-tls", tempoName)
}

func cookieSecretName(tempoName string) string {
	return fmt.Sprintf("tempo-%s-cookie-proxy", tempoName)
}

func proxyInitArguments(serviceAccountName string) []string {
	return []string{
		fmt.Sprintf("--cookie-secret-file=%s/%s", oauthProxySecretMountPath, sessionSecretKey),
		fmt.Sprintf("--https-address=:%d", manifestutils.OAuthProxyPort),
		fmt.Sprintf("--openshift-service-account=%s", serviceAccountName),
		"--provider=openshift",
		fmt.Sprintf("--tls-cert=%s/tls.crt", tlsProxyPath),
		fmt.Sprintf("--tls-key=%s/tls.key", tlsProxyPath),
		fmt.Sprintf("--upstream=http://localhost:%d", manifestutils.PortJaegerUI),
	}
}

func oAuthProxyContainer(
	tempo string,
	serviceAccountName string,
	authSpec *v1alpha1.JaegerQueryAuthenticationSpec,
	oauthProxyImage string,
) corev1.Container {
	args := proxyInitArguments(serviceAccountName)

	if len(strings.TrimSpace(authSpec.SAR)) > 0 {
		args = append(args, fmt.Sprintf("--openshift-sar=%s", authSpec.SAR))
	}

	resources := authSpec.Resources
	if resources == nil {
		resources = &corev1.ResourceRequirements{}
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
		VolumeMounts: []corev1.VolumeMount{{
			MountPath: tlsProxyPath,
			Name:      getTLSSecretNameForFrontendService(tempo),
		},

			{
				MountPath: oauthProxySecretMountPath,
				Name:      cookieSecretName(tempo),
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

func generateProxySecret() (string, error) {
	randomBytes := make([]byte, minBytesRequiredByCookieValue)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(randomBytes), nil
}

func getOAuthRedirectReference(routeName string) string {
	return fmt.Sprintf(
		`{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"%s"}}`,
		routeName)
}

package oauthproxy

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/operator-framework/operator-lib/proxy"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

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
func OAuthServiceAccount(params manifestutils.Params) *corev1.ServiceAccount {
	labels := manifestutils.ComponentLabels(manifestutils.QueryFrontendComponentName, params.Tempo.Name)
	annotations := map[string]string{
		serviceAccountRedirectAnnotation: getOAuthRedirectReference(naming.Name(manifestutils.QueryFrontendComponentName, params.Tempo.Name)),
	}
	if params.StorageParams.S3 != nil && params.StorageParams.S3.ShortLived != nil {
		awsAnnotations := manifestutils.S3AWSSTSAnnotations(*params.StorageParams.S3.ShortLived)
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

// PatchRouteForOauthProxy a modified route to use re-encrypt.
func PatchRouteForOauthProxy(route *routev1.Route) { // point route to the oauth proxy
	route.Spec.TLS = &routev1.TLSConfig{Termination: routev1.TLSTerminationReencrypt}
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

// PatchPodSpecForOauthProxy returns a modified PodSpec with the oauth sidecar container and the right service account.
func PatchPodSpecForOauthProxy(params Params, podSpec *corev1.PodSpec,

) {
	oauthProxyImage := params.ProxyImage
	if oauthProxyImage == "" {
		oauthProxyImage = params.ProjectConfig.DefaultImages.OauthProxy
	}

	if params.OverrideServiceAccount {
		podSpec.ServiceAccountName = naming.Name(manifestutils.QueryFrontendComponentName, params.TempoMeta.Name)
	}

	certsVolumeName := getTLSSecretNameForFrontendService(params.TempoMeta.Name)
	if !manifestutils.ContainsVolume(podSpec.Volumes, certsVolumeName) {
		podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
			Name: getTLSSecretNameForFrontendService(params.TempoMeta.Name),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: getTLSSecretNameForFrontendService(params.TempoMeta.Name),
				},
			},
		})
	}

	cookieVolumeName := cookieSecretName(params.TempoMeta.Name)
	if !manifestutils.ContainsVolume(podSpec.Volumes, cookieVolumeName) {
		podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
			Name: cookieSecretName(params.TempoMeta.Name),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: cookieSecretName(params.TempoMeta.Name),
				},
			},
		})
	}

	removePortFromContainer(params.ContainerName, params.Port, podSpec)

	podSpec.Containers = append(podSpec.Containers,
		oAuthProxyContainer(
			params.TempoMeta.Name,
			podSpec.ServiceAccountName,
			params.AuthSpec,
			oauthProxyImage,
			params.ContainerName,
			params.HTTPPort,
			params.HTTPSPort,
			params.Port,
		),
	)
}

func removePortFromContainer(containerName string, port corev1.ContainerPort, podSpec *corev1.PodSpec) {
	i, err := manifestutils.FindContainerIndex(podSpec, containerName)
	if err != nil {
		return
	}
	// Find the Port
	var newPorts []corev1.ContainerPort

	containers := podSpec.Containers

	if containers[i].Ports != nil {
		for _, containerPort := range containers[i].Ports {
			if containerPort.Name != port.Name || containerPort.ContainerPort != port.ContainerPort {
				newPorts = append(newPorts, containerPort)
			}
		}
	}
	containers[i].Ports = newPorts
}

func getTLSSecretNameForFrontendService(tempoName string) string {
	return fmt.Sprintf("%s-ui-oauth-proxy-tls", tempoName)
}

func cookieSecretName(tempoName string) string {
	return fmt.Sprintf("tempo-%s-cookie-proxy", tempoName)
}

func proxyInitArguments(serviceAccountName string, originalPort int32, proxyPort int32, proxyPortHTTP int32) []string {
	return []string{
		fmt.Sprintf("--cookie-secret-file=%s/%s", oauthProxySecretMountPath, sessionSecretKey),
		fmt.Sprintf("--https-address=:%d", proxyPort),
		fmt.Sprintf("--http-address=:%d", proxyPortHTTP),

		fmt.Sprintf("--openshift-service-account=%s", serviceAccountName),
		"--provider=openshift",
		fmt.Sprintf("--tls-cert=%s/tls.crt", tlsProxyPath),
		fmt.Sprintf("--tls-key=%s/tls.key", tlsProxyPath),
		fmt.Sprintf("--upstream=http://localhost:%d", originalPort),
	}
}

func oAuthProxyContainer(
	tempo string,
	serviceAccountName string,
	authSpec v1alpha1.OAuthAuthenticationSpec,
	oauthProxyImage string,
	containerName string,
	oauthProxyHTTPPort int32,
	oauthProxyHTTPSPort int32,
	containerPort corev1.ContainerPort) corev1.Container {

	originalPort := containerPort.ContainerPort
	containerPort.ContainerPort = oauthProxyHTTPSPort

	args := proxyInitArguments(serviceAccountName, originalPort, oauthProxyHTTPSPort, oauthProxyHTTPPort)

	if len(strings.TrimSpace(authSpec.SAR)) > 0 {
		args = append(args, fmt.Sprintf("--openshift-sar=%s", authSpec.SAR))
	}

	resources := authSpec.Resources
	if resources == nil {
		resources = &corev1.ResourceRequirements{}
	}

	return corev1.Container{
		Image: oauthProxyImage,
		Name:  fmt.Sprintf("%s-oauth-proxy", containerName),
		Args:  args,
		Ports: []corev1.ContainerPort{containerPort},
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
					Port:   intstr.FromString(containerPort.Name),
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

// IsOauthEnabled return true if oauth is enabled.
func IsOauthEnabled(spec *v1alpha1.OAuthAuthenticationSpec) bool {
	return spec != nil && spec.Enabled

}

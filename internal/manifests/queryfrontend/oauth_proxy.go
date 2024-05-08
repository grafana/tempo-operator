package queryfrontend

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
)

func generateProxySecret() (string, error) {
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(randomBytes), nil
}

func getOAuthRedirectReference(tempo v1alpha1.TempoStack) string {
	return fmt.Sprintf(
		`{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"%s"}}`,
		naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name))
}

// OAuthProxy returns a service account representing a client in the context of the OAuth Proxy.
func oauthServiceAccount(tempo v1alpha1.TempoStack) *corev1.ServiceAccount {
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
				"serviceaccounts.openshift.io/oauth-redirectreference.primary": getOAuthRedirectReference(tempo),
			},
		},
	}
}

func getTLSSecretNameForFrontendService(tempoName string) string {
	return fmt.Sprintf("%s-ui-oauth-proxy-tls", tempoName)
}

func patchRoute(route *routev1.Route) *routev1.Route { // point route to the oauth proxy
	route.Spec.TLS = &routev1.TLSConfig{Termination: routev1.TLSTerminationReencrypt}
	route.Spec.Port.TargetPort = intstr.FromString(manifestutils.OAuthProxyPortName)
	return route
}
func oauthCookieSessionSecret(tempo v1alpha1.TempoStack) (*corev1.Secret, error) {
	sessionSecret, err := generateProxySecret()

	if err != nil {
		return nil, err
	}

	labels := manifestutils.ComponentLabels(manifestutils.OAuthProxyPortName, tempo.Name)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cookieSecretName(tempo),
			Labels:    labels,
			Namespace: tempo.Namespace,
		},
		Data: map[string][]byte{
			sessionSecretKey: []byte(sessionSecret),
		},
	}, nil
}

func cookieSecretName(tempo v1alpha1.TempoStack) string {
	return fmt.Sprintf("tempo-%s-cookie-proxy", tempo.Name)
}

func proxyInitArguments(tempo v1alpha1.TempoStack) []string {

	return []string{
		fmt.Sprintf("--cookie-secret-file=%s/%s", oauthProxySecretMountPath, sessionSecretKey),
		fmt.Sprintf("--https-address=:%d", manifestutils.OAuthProxyPort),
		fmt.Sprintf("--openshift-service-account=%s", naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name)),
		"--provider=openshift",
		fmt.Sprintf("--tls-cert=%s/tls.crt", tlsProxyPath),
		fmt.Sprintf("--tls-key=%s/tls.key", tlsProxyPath),
		fmt.Sprintf("--upstream=http://localhost:%d", manifestutils.PortJaegerUI),
	}
}

func patchDeploymentForOauthProxy(params manifestutils.Params, dep *v1.Deployment) {
	tempo := params.Tempo
	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: getTLSSecretNameForFrontendService(tempo.Name),
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: getTLSSecretNameForFrontendService(tempo.Name),
			},
		},
	})
	dep.Spec.Template.Spec.ServiceAccountName = naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name)
	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, oAuthProxyContainer(params))
	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: cookieSecretName(tempo),
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: cookieSecretName(tempo),
			},
		},
	})
}

func oAuthProxyContainer(params manifestutils.Params) corev1.Container {
	tempo := params.Tempo
	args := proxyInitArguments(tempo)

	if len(strings.TrimSpace(tempo.Spec.Template.QueryFrontend.JaegerQuery.Oauth.SAR)) > 0 {
		args = append(args, fmt.Sprintf("--openshift-sar=%s", tempo.Spec.Template.QueryFrontend.JaegerQuery.Oauth.SAR))
	}

	oauthProxyImage := tempo.Spec.Images.OauthProxy
	if oauthProxyImage == "" {
		oauthProxyImage = params.CtrlConfig.DefaultImages.OauthProxy
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
			Name:      getTLSSecretNameForFrontendService(tempo.Name),
		},

			{
				MountPath: oauthProxySecretMountPath,
				Name:      cookieSecretName(tempo),
			},
		},
		Resources: resources(tempo),
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

func patchQueryFrontEndService(services []*corev1.Service, tempo v1alpha1.TempoStack) {

	var service *corev1.Service
	serviceName := naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name)
	for _, svc := range services {
		if svc.Name == serviceName {
			service = svc
			break
		}
	}

	if service == nil {
		return
	}

	if service.Annotations == nil {
		service.Annotations = make(map[string]string)
	}

	service.Annotations["service.beta.openshift.io/serving-cert-secret-name"] = getTLSSecretNameForFrontendService(tempo.Name)

	service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
		Name:       manifestutils.OAuthProxyPortName,
		Port:       manifestutils.OAuthProxyPort,
		TargetPort: intstr.FromString(manifestutils.OAuthProxyPortName),
	})
}

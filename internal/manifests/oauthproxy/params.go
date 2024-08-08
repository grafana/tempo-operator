package oauthproxy

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

// Params for patching a PodSpec and inject oauth proxy.
type Params struct {
	TempoMeta              metav1.ObjectMeta
	AuthSpec               v1alpha1.OAuthAuthenticationSpec
	ProjectConfig          configv1alpha1.ProjectConfig
	Port                   corev1.ContainerPort
	ProxyImage             string
	ContainerName          string
	OverrideServiceAccount bool
	HTTPPort               int32
	HTTPSPort              int32
	TLSSecretName          string
}

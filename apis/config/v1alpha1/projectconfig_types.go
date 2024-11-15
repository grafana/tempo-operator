package v1alpha1

import (
	"encoding/json"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
)

const (
	// EnvRelatedImageTempo contains the name of the environment variable where the tempo image location is stored.
	EnvRelatedImageTempo = "RELATED_IMAGE_TEMPO"

	// EnvRelatedImageJaegerQuery contains the name of the environment variable where the jaegerQuery image location is stored.
	EnvRelatedImageJaegerQuery = "RELATED_IMAGE_JAEGER_QUERY"

	// EnvRelatedImageTempoQuery contains the name of the environment variable where the tempoQuery image location is stored.
	EnvRelatedImageTempoQuery = "RELATED_IMAGE_TEMPO_QUERY"

	// EnvRelatedImageTempoGateway contains the name of the environment variable where the tempoGateway image location is stored.
	EnvRelatedImageTempoGateway = "RELATED_IMAGE_TEMPO_GATEWAY"

	// EnvRelatedImageTempoGatewayOpa contains the name of the environment variable where the tempoGatewayOpa image location is stored.
	EnvRelatedImageTempoGatewayOpa = "RELATED_IMAGE_TEMPO_GATEWAY_OPA"

	// EnvRelatedImageOauthProxy contains the name of the environment variable where the oauth-proxy image location is stored.
	EnvRelatedImageOauthProxy = "RELATED_IMAGE_OAUTH_PROXY"
)

// ImagesSpec defines the image for each container.
type ImagesSpec struct {
	// Tempo defines the tempo container image.
	//
	// +optional
	Tempo string `json:"tempo,omitempty" yaml:"tempo,omitempty"`

	// TempoQuery defines the tempo-query container image.
	//
	// +optional
	TempoQuery string `json:"tempoQuery,omitempty" yaml:"tempoQuery,omitempty"`

	// JaegerQuery defines the tempo-query container image.
	//
	// +optional
	JaegerQuery string `json:"jaegerQuery,omitempty" yaml:"jaegerQuery,omitempty"`

	// TempoGateway defines the tempo-gateway container image.
	//
	// +optional
	TempoGateway string `json:"tempoGateway,omitempty" yaml:"tempoGateway,omitempty"`

	// TempoGatewayOpa defines the OPA sidecar container for TempoGateway.
	//
	// +optional
	TempoGatewayOpa string `json:"tempoGatewayOpa,omitempty" yaml:"tempoGatewayOpa,omitempty"`

	// OauthProxy defines the oauth proxy image used to protect the jaegerUI on single tenant.
	//
	// +optional
	OauthProxy string `json:"oauthProxy,omitempty" yaml:"oauthProxy,omitempty"`
}

// BuiltInCertManagement is the configuration for the built-in facility to generate and rotate
// TLS client and serving certificates for all Tempo services and internal clients. All necessary
// secrets and configmaps for protecting the internal components will be created if this option is enabled.
type BuiltInCertManagement struct {
	// CACertValidity defines the total duration of the CA certificate validity.
	CACertValidity Duration `json:"caValidity,omitempty" yaml:"caValidity,omitempty"`
	// CACertRefresh defines the duration of the CA certificate validity until a rotation
	// should happen. It can be set up to 80% of CA certificate validity or equal to the
	// CA certificate validity. Latter should be used only for rotating only when expired.
	CACertRefresh Duration `json:"caRefresh,omitempty" yaml:"caRefresh,omitempty"`
	// CertValidity defines the total duration of the validity for all Tempo certificates.
	CertValidity Duration `json:"certValidity,omitempty" yaml:"certValidity,omitempty"`
	// CertRefresh defines the duration of the certificate validity until a rotation
	// should happen. It can be set up to 80% of certificate validity or equal to the
	// certificate validity. Latter should be used only for rotating only when expired.
	// The refresh is applied to all Tempo certificates at once.
	CertRefresh Duration `json:"certRefresh,omitempty" yaml:"certRefresh,omitempty"`
	// Enabled defines to flag to enable/disable built-in certificate management feature gate.
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

// Duration is a wrapper around metav1.Duration to support parsing duration from a string.
// Note that ProjectConfig allows setting times like this:
//
//	caValidity: 43830h
//
// Since no quotes are used, Golang will interpret 43830 as an integer and 'h' as a kind of scalar.
// Not a string. To avoid breaking changes, we need to support this format. That's why we need
// to implement custom unmarshalling logic.
type Duration struct {
	metav1.Duration `json:",inline" yaml:",inline"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		parsed, err := time.ParseDuration(str)
		if err != nil {
			return err
		}
		d.Duration = metav1.Duration{Duration: parsed}
		return nil
	}
	return json.Unmarshal(b, &d.Duration)
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		parsed, err := time.ParseDuration(str)
		if err != nil {
			return err
		}
		d.Duration = metav1.Duration{Duration: parsed}
		return nil
	}
	return unmarshal(&d.Duration)
}

// OpenShiftFeatureGates is the supported set of all operator features gates on OpenShift.
type OpenShiftFeatureGates struct {
	// ServingCertsService enables OpenShift service-ca annotations on the TempoStack
	// to use the in-platform CA and generate a TLS cert/key pair per service for
	// in-cluster data-in-transit encryption.
	// More details: https://docs.openshift.com/container-platform/latest/security/certificate_types_descriptions/service-ca-certificates.html
	//
	// Currently is only used in two cases:
	//   - If gateway is enabled, it will be used by the gateway component
	//   - If the gateway is disabled and TLS is enabled on the distributor but no caName and certName are specified
	ServingCertsService bool `json:"servingCertsService,omitempty" yaml:"servingCertsService,omitempty"`

	// OpenShiftRoute enables creating OpenShift Route objects.
	// More details: https://docs.openshift.com/container-platform/latest/networking/understanding-networking.html
	OpenShiftRoute bool `json:"openshiftRoute,omitempty" yaml:"openshiftRoute,omitempty"`

	// BaseDomain is used internally for redirect URL in gateway OpenShift auth mode.
	// If empty the operator automatically derives the domain from the cluster.
	BaseDomain string `json:"baseDomain,omitempty" yaml:"baseDomain,omitempty"`

	// ClusterTLSPolicy enables usage of TLS policies set in the API Server.
	// More details: https://docs.openshift.com/container-platform/4.11/security/tls-security-profiles.html
	ClusterTLSPolicy bool `json:"clusterTLSPolicy,omitempty" yaml:"clusterTLSPolicy,omitempty"`

	// OauthProxy define options for the oauth proxy feature.
	OauthProxy OauthProxyFeatureGates `json:"oAuthProxy,omitempty" yaml:"oAuthProxy,omitempty"`
}

// TLSProfileType is a TLS security profile based on the Mozilla definitions:
// https://wiki.mozilla.org/Security/Server_Side_TLS
type TLSProfileType string

const (
	// TLSProfileOldType is a TLS security profile based on:
	// https://wiki.mozilla.org/Security/Server_Side_TLS#Old_backward_compatibility
	TLSProfileOldType TLSProfileType = "Old"
	// TLSProfileIntermediateType is a TLS security profile based on:
	// https://wiki.mozilla.org/Security/Server_Side_TLS#Intermediate_compatibility_.28default.29
	TLSProfileIntermediateType TLSProfileType = "Intermediate"
	// TLSProfileModernType is a TLS security profile based on:
	// https://wiki.mozilla.org/Security/Server_Side_TLS#Modern_compatibility
	TLSProfileModernType TLSProfileType = "Modern"
)

// MetricsFeatureGates configures metrics and alerts of the operator.
type MetricsFeatureGates struct {
	// CreateServiceMonitors defines whether the operator should install ServiceMonitors
	// to scrape metrics of the operator.
	CreateServiceMonitors bool `json:"createServiceMonitors,omitempty" yaml:"createServiceMonitors,omitempty"`

	// CreatePrometheusRules defines whether the operator should install PrometheusRules
	// to receive alerts about the operator.
	CreatePrometheusRules bool `json:"createPrometheusRules,omitempty" yaml:"createPrometheusRules,omitempty"`
}

// ObservabilityFeatureGates configures observability of the operator.
type ObservabilityFeatureGates struct {
	// Metrics configures metrics of the operator.
	Metrics MetricsFeatureGates `json:"metrics,omitempty" yaml:"metrics,omitempty"`
}

// OauthProxyFeatureGates configures oauth proxy options.
type OauthProxyFeatureGates struct {
	// OAuthProxyEnabled is used internally for enable by default the oauth proxy for the UI when multi-tenancy is disabled.
	DefaultEnabled bool `json:"defaultEnabled,omitempty" yaml:"defaultEnabled,omitempty"`
}

// FeatureGates is the supported set of all operator feature gates.
type FeatureGates struct {
	// OpenShift contains a set of feature gates supported only on OpenShift.
	OpenShift OpenShiftFeatureGates `json:"openshift,omitempty" yaml:"openshift,omitempty"`

	// BuiltInCertManagement enables the built-in facility for generating and rotating
	// TLS client and serving certificates for the communication between ingesters and distributors and also between
	// query and query-frontend, In detail all internal Tempo HTTP and GRPC communication is lifted
	// to require mTLS.
	// In addition each service requires a configmap named as the MicroService CR with the
	// suffix `-ca-bundle`, e.g. `tempo-dev-ca-bundle` and the following data:
	// - `service-ca.crt`: The CA signing the service certificate in `tls.crt`.
	// All necessary secrets and configmaps for protecting the internal components will be created if this
	// option is enabled.
	BuiltInCertManagement BuiltInCertManagement `json:"builtInCertManagement,omitempty" yaml:"builtInCertManagement,omitempty"`
	// HTTPEncryption enables TLS encryption for all HTTP TempoStack components.
	// Each HTTP component requires a secret, the name should be the name of the component with the
	// suffix `-mtls` and prefix by the TempoStack name e.g `tempo-dev-distributor-mtls`.
	// It should contains the following data:
	// - `tls.crt`: The TLS server side certificate.
	// - `tls.key`: The TLS key for server-side encryption.
	// In addition each service requires a configmap named as the TempoStack CR with the
	// suffix `-ca-bundle`, e.g. `tempo-dev-ca-bundle` and the following data:
	// - `service-ca.crt`: The CA signing the service certificate in `tls.crt`.
	// This will protect all internal communication between the distributors and ingestors and also
	// between ingestor and queriers, and between the queriers and the query-frontend component
	//
	// If BuiltInCertManagement is enabled, you don't need to create this secrets manually.
	//
	// Some considerations when enable mTLS:
	// - If JaegerUI is enabled, it won't be protected by mTLS as it will be considered a public facing
	// component.
	// - If JaegerUI is not enabled, HTTP Tempo API won´t be protected, this will be considered
	// public faced component.
	// - If Gateway is enabled, all comunications between the gateway and the tempo components will be protected
	// by mTLS, and the Gateway itself won´t be, as it will be the only public face component.
	HTTPEncryption bool `json:"httpEncryption,omitempty" yaml:"httpEncryption,omitempty"`
	// GRPCEncryption enables TLS encryption for all GRPC TempoStack services.
	// Each GRPC component requires a secret, the name should be the name of the component with the
	// suffix `-mtls` and prefix by the TempoStack name e.g `tempo-dev-distributor-mtls`.
	// It should contains the following data:
	// - `tls.crt`: The TLS server side certificate.
	// - `tls.key`: The TLS key for server-side encryption.
	// In addition each service requires a configmap named as the TempoStack CR with the
	// suffix `-ca-bundle`, e.g. `tempo-dev-ca-bundle` and the following data:
	// - `service-ca.crt`: The CA signing the service certificate in `tls.crt`.
	// This will protect all internal communication between the distributors and ingestors and also
	// between ingestor and queriers, and between the queriers and the query-frontend component.
	//
	//
	// If BuiltInCertManagement is enabled, you don't need to create this secrets manually.
	//
	// Some considerations when enable mTLS:
	// - If JaegerUI is enabled, it won´t be protected by mTLS as it will be considered a public face
	// component.
	// - If Gateway is enabled, all comunications between the gateway and the tempo components will be protected
	// by mTLS, and the Gateway itself won´t be, as it will be the only public face component.
	GRPCEncryption bool `json:"grpcEncryption,omitempty" yaml:"grpcEncryption,omitempty"`

	// TLSProfile allows to chose a TLS security profile. Enforced
	// when using HTTPEncryption or GRPCEncryption.
	TLSProfile string `json:"tlsProfile,omitempty" yaml:"tlsProfile,omitempty"`

	// PrometheusOperator defines whether the Prometheus Operator CRD exists in the cluster.
	// This CRD is part of prometheus-operator.
	PrometheusOperator bool `json:"prometheusOperator,omitempty" yaml:"prometheusOperator,omitempty"`

	// Observability configures observability features of the operator.
	Observability ObservabilityFeatureGates `json:"observability,omitempty" yaml:"observability,omitempty"`

	// GrafanaOperator defines whether the Grafana Operator CRD exists in the cluster.
	// This CRD is part of grafana-operator.
	GrafanaOperator bool `json:"grafanaOperator,omitempty" yaml:"grafanaOperator,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ProjectConfig is the Schema for the projectconfigs API.
type ProjectConfig struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`

	// LeaderElection is the LeaderElection config to be used when configuring
	// the manager.Manager leader election
	LeaderElection *configv1alpha1.LeaderElectionConfiguration `json:"leaderElection,omitempty" yaml:"leaderElection,omitempty"`

	// Metrics contains the controller metrics configuration
	Metrics ControllerMetrics `json:"metrics,omitempty" yaml:"metrics,omitempty"`

	// Health contains the controller health configuration
	Health ControllerHealth `json:"health,omitempty" yaml:"health,omitempty"`

	// Webhook contains the controllers webhook configuration
	Webhook ControllerWebhook `json:"webhook,omitempty" yaml:"webhook,omitempty"`

	// The images are read from environment variables and not from the configuration file
	DefaultImages ImagesSpec `json:"defaultImages,omitempty" yaml:"defaultImages,omitempty"`

	Gates FeatureGates `json:"featureGates,omitempty" yaml:"featureGates,omitempty"`

	// Distribution defines the operator distribution name.
	Distribution string `json:"distribution" yaml:"distribution"`
}

// ControllerMetrics is the metrics configuration for the controller.
type ControllerMetrics struct {
	// BindAddress is the TCP address that the controller should bind to
	// for serving prometheus metrics.
	// It can be set to "0" to disable the metrics serving.
	BindAddress string `json:"bindAddress,omitempty" yaml:"bindAddress,omitempty"`
}

// ControllerHealth is the health configuration for the controller.
type ControllerHealth struct {
	// HealthProbeBindAddress is the TCP address that the controller should bind to
	// for serving health probes
	// It can be set to "0" or "" to disable serving the health probe.
	HealthProbeBindAddress string `json:"healthProbeBindAddress,omitempty" yaml:"healthProbeBindAddress,omitempty"`

	// ReadinessEndpointName, defaults to "readyz"
	ReadinessEndpointName string `json:"readinessEndpointName,omitempty" yaml:"readinessEndpointName,omitempty"`

	// LivenessEndpointName, defaults to "healthz"
	LivenessEndpointName string `json:"livenessEndpointName,omitempty" yaml:"livenessEndpointName,omitempty"`
}

// ControllerWebhook is the webhook configuration for the controller.
type ControllerWebhook struct {
	// Port is the port that the webhook server serves at.
	// It is used to set webhook.Server.Port.
	Port *int `json:"port,omitempty" yaml:"port,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ProjectConfig{})
}

// DefaultProjectConfig returns the default operator config.
func DefaultProjectConfig() ProjectConfig {
	return ProjectConfig{
		DefaultImages: ImagesSpec{
			Tempo:           os.Getenv(EnvRelatedImageTempo),
			JaegerQuery:     os.Getenv(EnvRelatedImageJaegerQuery),
			TempoQuery:      os.Getenv(EnvRelatedImageTempoQuery),
			TempoGateway:    os.Getenv(EnvRelatedImageTempoGateway),
			TempoGatewayOpa: os.Getenv(EnvRelatedImageTempoGatewayOpa),
			OauthProxy:      os.Getenv(EnvRelatedImageOauthProxy),
		},
		Gates: FeatureGates{
			TLSProfile: string(TLSProfileModernType),
		},
	}
}

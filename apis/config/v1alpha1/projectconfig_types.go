package v1alpha1

import (
	"fmt"

	dockerparser "github.com/novln/docker-parser"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

// ImagesSpec defines the image for each container.
type ImagesSpec struct {
	// Tempo defines the tempo container image.
	//
	// +optional
	Tempo string `json:"tempo,omitempty"`

	// TempoQuery defines the tempo-query container image.
	//
	// +optional
	TempoQuery string `json:"tempoQuery,omitempty"`

	// TempoGateway defines the tempo-gateway container image.
	//
	// +optional
	TempoGateway string `json:"tempoGateway,omitempty"`

	// TempoGatewayOpa defines the OPA sidecar container for TempoGateway.
	//
	// +optional
	TempoGatewayOpa string `json:"tempoGatewayOpa,omitempty"`
}

// BuiltInCertManagement is the configuration for the built-in facility to generate and rotate
// TLS client and serving certificates for all Tempo services and internal clients except
// for the tempo-gateway.
type BuiltInCertManagement struct {
	// CACertValidity defines the total duration of the CA certificate validity.
	CACertValidity metav1.Duration `json:"caValidity,omitempty"`
	// CACertRefresh defines the duration of the CA certificate validity until a rotation
	// should happen. It can be set up to 80% of CA certificate validity or equal to the
	// CA certificate validity. Latter should be used only for rotating only when expired.
	CACertRefresh metav1.Duration `json:"caRefresh,omitempty"`
	// CertValidity defines the total duration of the validity for all Tempo certificates.
	CertValidity metav1.Duration `json:"certValidity,omitempty"`
	// CertRefresh defines the duration of the certificate validity until a rotation
	// should happen. It can be set up to 80% of certificate validity or equal to the
	// certificate validity. Latter should be used only for rotating only when expired.
	// The refresh is applied to all Tempo certificates at once.
	CertRefresh metav1.Duration `json:"certRefresh,omitempty"`
	// Enabled defines to flag to enable/disable built-in certificate management feature gate.
	Enabled bool `json:"enabled,omitempty"`
}

// OpenShiftFeatureGates is the supported set of all operator features gates on OpenShift.
type OpenShiftFeatureGates struct {
	// ServingCertsService enables OpenShift service-ca annotations on the TempoStack gateway service only
	// to use the in-platform CA and generate a TLS cert/key pair per service for
	// in-cluster data-in-transit encryption.
	// More details: https://docs.openshift.com/container-platform/latest/security/certificate_types_descriptions/service-ca-certificates.html
	ServingCertsService bool `json:"servingCertsService,omitempty"`

	// OpenShiftRoute enables creating OpenShift Route objects.
	// More details: https://docs.openshift.com/container-platform/latest/networking/understanding-networking.html
	OpenShiftRoute bool `json:"openshiftRoute,omitempty"`

	// BaseDomain is used internally for redirect URL in gateway OpenShift auth mode.
	// If empty the operator automatically derives the domain from the cluster.
	BaseDomain string `json:"baseDomain,omitempty"`

	// ClusterTLSPolicy enables usage of TLS policies set in the API Server.
	// More details: https://docs.openshift.com/container-platform/4.11/security/tls-security-profiles.html
	ClusterTLSPolicy bool
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

// FeatureGates is the supported set of all operator feature gates.
type FeatureGates struct {
	// OpenShift contains a set of feature gates supported only on OpenShift.
	OpenShift OpenShiftFeatureGates `json:"openshift,omitempty"`

	// BuiltInCertManagement enables the built-in facility for generating and rotating
	// TLS client and serving certificates for the communication between ingesters and distributors and also between
	// query and queryfrontend, In detail all internal Tempo HTTP and GRPC communication is lifted
	// to require mTLS.
	// In addition each service requires a configmap named as the MicroService CR with the
	// suffix `-ca-bundle`, e.g. `tempo-dev-ca-bundle` and the following data:
	// - `service-ca.crt`: The CA signing the service certificate in `tls.crt`.
	BuiltInCertManagement BuiltInCertManagement `json:"builtInCertManagement,omitempty"`
	// HTTPEncryption enables TLS encryption for all HTTP TempoStack services.
	// Each HTTP service requires a secret named as the service with the following data:
	// - `tls.crt`: The TLS server side certificate.
	// - `tls.key`: The TLS key for server-side encryption.
	// In addition each service requires a configmap named as the TempoStack CR with the
	// suffix `-ca-bundle`, e.g. `tempo-dev-ca-bundle` and the following data:
	// - `service-ca.crt`: The CA signing the service certificate in `tls.crt`.
	// This will protect all internal communication between the distributors and ingestors and also
	// between ingestor and queriers, and between the queriers and the query-frontend component
	// The only component remains unprotected is the tempo-query (jaeger query UI).
	HTTPEncryption bool `json:"httpEncryption,omitempty"`
	// GRPCEncryption enables TLS encryption for all GRPC TempoStack services.
	// Each GRPC service requires a secret named as the service with the following data:
	// - `tls.crt`: The TLS server side certificate.
	// - `tls.key`: The TLS key for server-side encryption.
	// In addition each service requires a configmap named as the TempoStack CR with the
	// suffix `-ca-bundle`, e.g. `tempo-dev-ca-bundle` and the following data:
	// - `service-ca.crt`: The CA signing the service certificate in `tls.crt`.
	// This will protect all internal communication between the distributors and ingestors and also
	// between ingestor and queriers, and between the queriers and the query-frontend component.
	// The only component remains unprotected is the tempo-query (jaeger query UI).
	GRPCEncryption bool `json:"grpcEncryption,omitempty"`

	// TLSProfile allows to chose a TLS security profile. Enforced
	// when using HTTPEncryption or GRPCEncryption.
	TLSProfile string `json:"tlsProfile,omitempty"`

	// ServiceMonitors enables the creation of ServiceMonitor instances for all components of the Tempo Stack.
	// Note: This feature requires prometheus-operator to be installed in the cluster.
	ServiceMonitors bool `json:"serviceMonitors,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ProjectConfig is the Schema for the projectconfigs API.
type ProjectConfig struct {
	metav1.TypeMeta `json:",inline"`
	// ControllerManagerConfigurationSpec returns the configurations for controllers
	cfg.ControllerManagerConfigurationSpec `json:",inline"`

	DefaultImages ImagesSpec `json:"images"`

	Gates FeatureGates `json:"featureGates,omitempty"`

	// Distribution defines the operator distribution name.
	Distribution string `json:"distribution"`
}

func init() {
	SchemeBuilder.Register(&ProjectConfig{})
}

// Validate validates the controller configuration (ProjectConfig).
func (c *ProjectConfig) Validate() error {
	switch c.Gates.TLSProfile {
	case string(TLSProfileOldType),
		string(TLSProfileIntermediateType),
		string(TLSProfileModernType):
		// valid setting
	default:
		return fmt.Errorf("invalid value '%s' for setting featureGates.tlsProfile (valid values: %s, %s and %s)", c.Gates.TLSProfile, TLSProfileOldType, TLSProfileIntermediateType, TLSProfileModernType)
	}

	if c.DefaultImages.Tempo != "" {
		_, err := dockerparser.Parse(c.DefaultImages.Tempo)
		if err != nil {
			return fmt.Errorf("invalid value '%s' for setting images.tempo", c.DefaultImages.Tempo)
		}
	}
	if c.DefaultImages.TempoQuery != "" {
		_, err := dockerparser.Parse(c.DefaultImages.TempoQuery)
		if err != nil {
			return fmt.Errorf("invalid value '%s' for setting images.tempoQuery", c.DefaultImages.TempoQuery)
		}
	}
	if c.DefaultImages.TempoGateway != "" {
		_, err := dockerparser.Parse(c.DefaultImages.TempoGateway)
		if err != nil {
			return fmt.Errorf("invalid value '%s' for setting images.tempoGateway", c.DefaultImages.TempoGateway)
		}
	}

	return nil
}

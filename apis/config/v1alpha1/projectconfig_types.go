package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"

	apiv1alpha1 "github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

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

	// GatewayRoute enables creating an OpenShift Route for the TempoStack
	// gateway to expose the service to public internet access.
	// More details: https://docs.openshift.com/container-platform/latest/networking/understanding-networking.html
	GatewayRoute bool `json:"gatewayRoute,omitempty"`
}

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
	// HTTPEncryption enables TLS encryption for all HTTP Microservices services.
	// Each HTTP service requires a secret named as the service with the following data:
	// - `tls.crt`: The TLS server side certificate.
	// - `tls.key`: The TLS key for server-side encryption.
	// In addition each service requires a configmap named as the Microservices CR with the
	// suffix `-ca-bundle`, e.g. `tempo-dev-ca-bundle` and the following data:
	// - `service-ca.crt`: The CA signing the service certificate in `tls.crt`.
	// This will protect all internal communication between the distributors and ingestors and also
	// between ingestor and queriers, and between the queriers and the query-frontend component
	// The only component remains unprotected is the tempo-query (jaeger query UI).
	HTTPEncryption bool `json:"httpEncryption,omitempty"`
	// GRPCEncryption enables TLS encryption for all GRPC Microservices services.
	// Each GRPC service requires a secret named as the service with the following data:
	// - `tls.crt`: The TLS server side certificate.
	// - `tls.key`: The TLS key for server-side encryption.
	// In addition each service requires a configmap named as the Microservices CR with the
	// suffix `-ca-bundle`, e.g. `tempo-dev-ca-bundle` and the following data:
	// - `service-ca.crt`: The CA signing the service certificate in `tls.crt`.
	// This will protect all internal communication between the distributors and ingestors and also
	// between ingestor and queriers, and between the queriers and the query-frontend component.
	// The only component remains unprotected is the tempo-query (jaeger query UI).
	GRPCEncryption bool `json:"grpcEncryption,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ProjectConfig is the Schema for the projectconfigs API.
type ProjectConfig struct {
	metav1.TypeMeta `json:",inline"`
	// ControllerManagerConfigurationSpec returns the configurations for controllers
	cfg.ControllerManagerConfigurationSpec `json:",inline"`

	DefaultImages apiv1alpha1.ImagesSpec `json:"images"`

	Gates FeatureGates `json:"featureGates,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ProjectConfig{})
}

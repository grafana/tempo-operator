package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// PodStatusMap defines the type for mapping pod status to pod name.
type PodStatusMap map[corev1.PodPhase][]string

// TLSSpec is the TLS configuration.
type TLSSpec struct {
	// Enabled defines if TLS is enabled.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enabled",order=1,xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	Enabled bool `json:"enabled"`

	// CA is the name of a ConfigMap containing a CA certificate (service-ca.crt).
	// It needs to be in the same namespace as the Tempo custom resource.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:io.kubernetes:ConfigMap",displayName="CA ConfigMap"
	CA string `json:"caName,omitempty"`

	// Cert is the name of a Secret containing a certificate (tls.crt) and private key (tls.key).
	// It needs to be in the same namespace as the Tempo custom resource.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:io.kubernetes:Secret",displayName="Certificate Secret"
	Cert string `json:"certName,omitempty"`

	// MinVersion defines the minimum acceptable TLS version.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Min TLS Version"
	MinVersion string `json:"minVersion,omitempty"`
}

// ExtraConfigSpec defines extra configurations for tempo that will be merged with the operator generated, configurations defined here
// has precedence and could override generated config.
type ExtraConfigSpec struct {
	// Tempo defines any extra Tempo configuration, which will be merged with the operator's generated Tempo configuration
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tempo Extra Configurations"
	Tempo apiextensionsv1.JSON `json:"tempo,omitempty"`
}

// JaegerQueryAuthenticationSpec defines options applied to proxy sidecar that controls the authentication of the jaeger UI.
type JaegerQueryAuthenticationSpec struct {
	// Defines if the authentication will be enabled for jaeger UI.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enabled",order=1,xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	Enabled bool `json:"enabled"`

	// SAR defines the SAR to be used in the oauth-proxy
	// default is "{"namespace": "<tempo_stack_namespace>", "resource": "pods", "verb": "get"}
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="SAR"
	SAR string `json:"sar,omitempty"`
	// Resources defines the compute resource requirements of the OAuth Proxy container.
	// The OAuth Proxy performs authentication and authorization of incoming requests to Jaeger UI when multi-tenancy is disabled.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resources",xDescriptors="urn:alm:descriptor:com.tectonic.ui:resourceRequirements"
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// CredentialMode represents the type of authentication used for accessing the object storage.
//
// +kubebuilder:validation:Enum=static;token;token-cco
type CredentialMode string

const (
	// CredentialModeStatic represents the usage of static, long-lived credentials stored in a Secret.
	// This is the default authentication mode and available for all supported object storage types.
	CredentialModeStatic CredentialMode = "static"
	// CredentialModeToken represents the usage of short-lived tokens retrieved from a credential source.
	// In this mode the static configuration does not contain credentials needed for the object storage.
	// Instead, they are generated during runtime using a service, which allows for shorter-lived credentials and
	// much more granular control. This authentication mode is not supported for all object storage types.
	CredentialModeToken CredentialMode = "token"
	// CredentialModeTokenCCO represents the usage of short-lived tokens retrieved from a credential source.
	// This mode is similar to CredentialModeToken, but instead of having a user-configured credential source,
	// it is configured by the environment and the operator relies on the Cloud Credential Operator to provide
	// a secret. This mode is only supported for certain object storage types in certain runtime environments.
	CredentialModeTokenCCO CredentialMode = "token-cco"
)

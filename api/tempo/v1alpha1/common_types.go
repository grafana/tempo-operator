package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// PodStatus is a short description of the status a Pod can be in.
type PodStatus string

const (
	// PodPending means the pod has been accepted by the system, but one or more of the containers
	// has not been started.
	PodPending PodStatus = "Pending"
	// PodRunning means the pod has been bound to a node and all of the containers have been started.
	// At least one container is still running or is in the process of being restarted.
	PodRunning PodStatus = "Running"
	// PodReady means the pod has been started and the readiness probe reports a successful status.
	PodReady PodStatus = "Ready"
	// PodSucceeded means that all containers in the pod have terminated in success, and will not be restarted.
	PodSucceeded PodStatus = "Succeeded"
	// PodFailed means that all containers in the pod have terminated, and at least one container has
	// terminated in a failure.
	PodFailed PodStatus = "Failed"
	// PodStatusUnknown is used when none of the other statuses apply or the information is not ready yet.
	PodStatusUnknown PodStatus = "Unknown"
)

// PodStatusMap defines the type for mapping pod status to pod name.
type PodStatusMap map[PodStatus][]string

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
	// If not set, the version is set based on feature gate tlsProfile or obtained from the cluster if openshift.clusterTLSPolicy is enabled.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Min TLS Version"
	MinVersion string `json:"minVersion,omitempty"`

	// CipherSuites defines the list of acceptable TLS cipher suites.
	//
	// If not set, the ciphers are set based on feature gate tlsProfile or obtained from the cluster if openshift.clusterTLSPolicy is enabled.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Cipher Suites"
	CipherSuites []string `json:"cipherSuites,omitempty"`
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

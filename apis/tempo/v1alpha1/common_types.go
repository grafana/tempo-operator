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

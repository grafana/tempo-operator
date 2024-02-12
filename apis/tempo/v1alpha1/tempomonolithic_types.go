package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TempoMonolithicSpec defines the desired state of TempoMonolithic.
type TempoMonolithicSpec struct {
	// Storage defines the storage configuration.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Storage"
	Storage *MonolithicStorageSpec `json:"storage,omitempty"`

	// Ingestion defines the trace ingestion configuration.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ingestion"
	Ingestion *MonolithicIngestionSpec `json:"ingestion,omitempty"`

	// JaegerUI defines the Jaeger UI configuration.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Jaeger UI"
	JaegerUI *MonolithicJaegerUISpec `json:"jaegerui,omitempty"`

	// ManagementState defines whether this instance is managed by the operator or self-managed.
	// Default: Managed.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Management State"
	Management ManagementStateType `json:"management,omitempty"`

	// Resources defines the compute resource requirements of the Tempo container.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resources",xDescriptors="urn:alm:descriptor:com.tectonic.ui:resourceRequirements"
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// ExtraConfig defines any extra (overlay) configuration of components.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Extra Configuration"
	ExtraConfig *ExtraConfigSpec `json:"extraConfig,omitempty"`
}

// MonolithicStorageSpec defines the storage for the Tempo deployment.
type MonolithicStorageSpec struct {
	// Traces defines the storage configuration for traces.
	//
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Traces"
	Traces MonolithicTracesStorageSpec `json:"traces"`
}

// MonolithicTracesStorageSpec defines the traces storage for the Tempo deployment.
type MonolithicTracesStorageSpec struct {
	// Backend defines the backend for storing traces.
	// Default: memory.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:default=memory
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Storage Backend"
	Backend MonolithicTracesStorageBackend `json:"backend"`

	// Size defines the size of the volume where traces are stored.
	// For in-memory storage, this defines the size of the tmpfs volume.
	// For persistent volume storage, this defines the size of the persistent volume.
	// For object storage, this defines the size of the persistent volume containing the Write-Ahead Log (WAL) of Tempo.
	// Default: 10Gi.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="10Gi"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Size",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	Size *resource.Quantity `json:"size,omitempty"`

	// S3 defines the configuration for Amazon S3.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Amazon S3"
	S3 *MonolithicTracesStorageS3Spec `json:"s3,omitempty"`

	// Azure defines the configuration for Azure Storage.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Azure Storage"
	Azure *MonolithicTracesObjectStorageSpec `json:"azure,omitempty"`

	// GCP defines the configuration for Google Cloud Storage.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Google Cloud Storage"
	GCS *MonolithicTracesObjectStorageSpec `json:"gcs,omitempty"`
}

// MonolithicTracesStorageBackend defines the backend storage for traces.
//
// +kubebuilder:validation:Enum=memory;pv;azure;gcs;s3
type MonolithicTracesStorageBackend string

const (
	// MonolithicTracesStorageBackendMemory defines storing traces in a tmpfs (in-memory filesystem).
	MonolithicTracesStorageBackendMemory MonolithicTracesStorageBackend = "memory"
	// MonolithicTracesStorageBackendPV defines storing traces in a Persistent Volume.
	MonolithicTracesStorageBackendPV MonolithicTracesStorageBackend = "pv"
	// MonolithicTracesStorageBackendAzure defines storing traces in Azure Storage.
	MonolithicTracesStorageBackendAzure MonolithicTracesStorageBackend = "azure"
	// MonolithicTracesStorageBackendGCS defines storing traces in Google Cloud Storage.
	MonolithicTracesStorageBackendGCS MonolithicTracesStorageBackend = "gcs"
	// MonolithicTracesStorageBackendS3 defines storing traces in Amazon S3.
	MonolithicTracesStorageBackendS3 MonolithicTracesStorageBackend = "s3"
)

// MonolithicTracesObjectStorageSpec defines object storage configuration.
type MonolithicTracesObjectStorageSpec struct {
	// Secret is the name of a Secret containing credentials for accessing object storage.
	// It needs to be in the same namespace as the TempoMonolithic custom resource.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Storage Secret",xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	Secret string `json:"secret"`
}

// MonolithicTracesStorageS3Spec defines the Amazon S3 configuration.
type MonolithicTracesStorageS3Spec struct {
	MonolithicTracesObjectStorageSpec `json:",inline"`

	// TLS defines the TLS configuration for Amazon S3.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="TLS"
	TLS *TLSSpec `json:"tls,omitempty"`
}

// MonolithicIngestionSpec defines the ingestion settings.
type MonolithicIngestionSpec struct {
	// OTLP defines the ingestion configuration for the OTLP protocol.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OTLP"
	OTLP *MonolithicIngestionOTLPSpec `json:"otlp,omitempty"`
}

// MonolithicIngestionOTLPSpec defines the settings for OTLP ingestion.
type MonolithicIngestionOTLPSpec struct {
	// GRPC defines the OTLP over gRPC configuration.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="gRPC"
	GRPC *MonolithicIngestionOTLPProtocolsGRPCSpec `json:"grpc,omitempty"`

	// HTTP defines the OTLP over HTTP configuration.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="HTTP"
	HTTP *MonolithicIngestionOTLPProtocolsHTTPSpec `json:"http,omitempty"`
}

// MonolithicIngestionOTLPProtocolsGRPCSpec defines the settings for OTLP ingestion over GRPC.
type MonolithicIngestionOTLPProtocolsGRPCSpec struct {
	// Enabled defines if OTLP over gRPC is enabled.
	// Default: enabled.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:default=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enabled",xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	Enabled bool `json:"enabled"`

	// TLS defines the TLS configuration for OTLP/gRPC ingestion.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="TLS"
	TLS *TLSSpec `json:"tls,omitempty"`
}

// MonolithicIngestionOTLPProtocolsHTTPSpec defines the settings for OTLP ingestion over HTTP.
type MonolithicIngestionOTLPProtocolsHTTPSpec struct {
	// Enabled defines if OTLP over HTTP is enabled.
	// Default: enabled.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:default=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enabled",xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	Enabled bool `json:"enabled"`

	// TLS defines the TLS configuration for OTLP/HTTP ingestion.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="TLS"
	TLS *TLSSpec `json:"tls,omitempty"`
}

// MonolithicJaegerUISpec defines the settings for the Jaeger UI.
type MonolithicJaegerUISpec struct {
	// Enabled defines if the Jaeger UI component should be created.
	//
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enabled",xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	Enabled bool `json:"enabled"`

	// Ingress defines the Ingress configuration for the Jaeger UI.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ingress"
	Ingress *MonolithicJaegerUIIngressSpec `json:"ingress,omitempty"`

	// Route defines the OpenShift route configuration for the Jaeger UI.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Route"
	Route *MonolithicJaegerUIRouteSpec `json:"route,omitempty"`

	// Resources defines the compute resource requirements of the Jaeger UI container.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resources",xDescriptors="urn:alm:descriptor:com.tectonic.ui:resourceRequirements"
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// MonolithicJaegerUIIngressSpec defines the settings for the Jaeger UI ingress.
type MonolithicJaegerUIIngressSpec struct {
	// Enabled defines if an Ingress object should be created for Jaeger UI.
	//
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enabled",xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	Enabled bool `json:"enabled"`

	// Annotations defines the annotations of the Ingress object.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Annotations"
	Annotations map[string]string `json:"annotations,omitempty"`

	// Host defines the hostname of the Ingress object.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Hostname"
	Host string `json:"host,omitempty"`

	// IngressClassName defines the name of an IngressClass cluster resource.
	// Defines which ingress controller serves this ingress resource.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ingress Class Name"
	IngressClassName *string `json:"ingressClassName,omitempty"`
}

// MonolithicJaegerUIRouteSpec defines the settings for the Jaeger UI route.
type MonolithicJaegerUIRouteSpec struct {
	// Enabled defines if a Route object should be created for Jaeger UI.
	//
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enabled",xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	Enabled bool `json:"enabled"`

	// Annotations defines the annotations of the Route object.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Annotations"
	Annotations map[string]string `json:"annotations,omitempty"`

	// Host defines the hostname of the Route object.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Hostname"
	Host string `json:"host,omitempty"`

	// Termination specifies the termination type.
	// Default: edge.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=edge
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="TLS Termination"
	Termination TLSRouteTerminationType `json:"termination,omitempty"`
}

// MonolithicComponentStatus defines the status of each component.
type MonolithicComponentStatus struct {
	// Tempo is a map of the pod status of the Tempo pods.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Tempo",xDescriptors="urn:alm:descriptor:com.tectonic.ui:podStatuses"
	Tempo PodStatusMap `json:"tempo"`
}

// TempoMonolithicStatus defines the observed state of TempoMonolithic.
type TempoMonolithicStatus struct {
	// Components provides summary of all Tempo pod status, grouped per component.
	//
	// +kubebuilder:validation:Optional
	Components MonolithicComponentStatus `json:"components,omitempty"`

	// Conditions of the Tempo deployment health.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:io.kubernetes.conditions"
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TempoMonolithic manages a Tempo deployment in monolithic mode.
//
// +operator-sdk:csv:customresourcedefinitions:displayName="TempoMonolithic",resources={{ConfigMap,v1},{Service,v1},{StatefulSet,v1},{Ingress,v1},{Route,v1}}
//
//nolint:godot
type TempoMonolithic struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TempoMonolithicSpec   `json:"spec,omitempty"`
	Status TempoMonolithicStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TempoMonolithicList contains a list of TempoMonolithic.
type TempoMonolithicList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TempoMonolithic `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TempoMonolithic{}, &TempoMonolithicList{})
}

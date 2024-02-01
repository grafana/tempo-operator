package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TempoMonolithicSpec defines the desired state of TempoMonolithic.
type TempoMonolithicSpec struct {
	// Storage defines the backend storage configuration
	//
	// +kubebuilder:validation:Optional
	Storage *MonolithicStorageSpec `json:"storage,omitempty"`

	// Ingestion defines the trace ingestion configuration
	//
	// +kubebuilder:validation:Optional
	Ingestion *MonolithicIngestionSpec `json:"ingestion,omitempty"`

	// JaegerUI defines the Jaeger UI configuration
	//
	// +kubebuilder:validation:Optional
	JaegerUI *MonolithicJaegerUISpec `json:"jaegerui,omitempty"`

	// ManagementState defines whether this instance is managed by the operator or self-managed
	//
	// +kubebuilder:validation:Optional
	Management ManagementStateType `json:"management,omitempty"`

	// Resources defines the compute resource requirements of Tempo.
	//
	// +kubebuilder:validation:Optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// ExtraConfig defines any extra (overlay) configuration for components
	//
	// +kubebuilder:validation:Optional
	ExtraConfig *ExtraConfigSpec `json:"extraConfig,omitempty"`
}

// MonolithicStorageSpec defines the storage for the Tempo deployment.
type MonolithicStorageSpec struct {
	// Traces defines the backend storage configuration for traces
	//
	// +kubebuilder:validation:Required
	Traces MonolithicTracesStorageSpec `json:"traces"`
}

// MonolithicTracesStorageSpec defines the traces storage for the Tempo deployment.
type MonolithicTracesStorageSpec struct {
	// Backend defines the backend for storing traces. Default: memory
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:default=memory
	Backend MonolithicTracesStorageBackend `json:"backend"`

	// Size defines the size of the volume where traces are stored.
	// For in-memory storage, this defines the size of the tmpfs volume.
	// For persistent volume storage, this defines the size of the persistent volume.
	// For object storage, this defines the size of the persistent volume containing the Write-Ahead Log (WAL) of Tempo.
	// Defaults to 10Gi.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="10Gi"
	Size *resource.Quantity `json:"size,omitempty"`

	// S3 defines the AWS S3 configuration
	//
	// +kubebuilder:validation:Optional
	S3 *MonolithicTracesStorageS3Spec `json:"s3,omitempty"`

	// Azure defines the Azure Storage configuration
	//
	// +kubebuilder:validation:Optional
	Azure *MonolithicTracesObjectStorageSpec `json:"azure,omitempty"`

	// GCP defines the Google Cloud Storage configuration
	//
	// +kubebuilder:validation:Optional
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
	// MonolithicTracesStorageBackendS3 defines storing traces in AWS S3.
	MonolithicTracesStorageBackendS3 MonolithicTracesStorageBackend = "s3"
)

// MonolithicTracesStorageWALSpec defines the write-ahead logging (WAL) configuration.
type MonolithicTracesStorageWALSpec struct {
	// Size defines the size of the Persistent Volume for storing the WAL. Defaults to 10Gi.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:default="10Gi"
	Size resource.Quantity `json:"size"`
}

// MonolithicTracesStoragePVSpec defines the Persistent Volume configuration.
type MonolithicTracesStoragePVSpec struct {
	// Size defines the size of the Persistent Volume for storing the traces. Defaults to 10Gi.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:default="10Gi"
	Size resource.Quantity `json:"size"`
}

// MonolithicTracesObjectStorageSpec defines object storage configuration.
type MonolithicTracesObjectStorageSpec struct {
	// secret is the name of a Secret containing credentials for accessing object storage.
	// It needs to be in the same namespace as the Tempo custom resource.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Secret string `json:"secret"`
}

// MonolithicTracesStorageS3Spec defines the AWS S3 configuration.
type MonolithicTracesStorageS3Spec struct {
	MonolithicTracesObjectStorageSpec `json:",inline"`

	// tls defines the TLS configuration for AWS S3.
	//
	// +kubebuilder:validation:Optional
	TLS *TLSSpec `json:"tls,omitempty"`
}

// MonolithicIngestionSpec defines the ingestion settings.
type MonolithicIngestionSpec struct {
	// OTLP defines the ingestion configuration for OTLP
	//
	// +kubebuilder:validation:Optional
	OTLP *MonolithicIngestionOTLPSpec `json:"otlp,omitempty"`
}

// MonolithicIngestionOTLPSpec defines the settings for OTLP ingestion.
type MonolithicIngestionOTLPSpec struct {
	// GRPC defines the OTLP/gRPC configuration
	//
	// +kubebuilder:validation:Optional
	GRPC *MonolithicIngestionOTLPProtocolsGRPCSpec `json:"grpc,omitempty"`

	// HTTP defines the OTLP/HTTP configuration
	//
	// +kubebuilder:validation:Optional
	HTTP *MonolithicIngestionOTLPProtocolsHTTPSpec `json:"http,omitempty"`
}

// MonolithicIngestionOTLPProtocolsGRPCSpec defines the settings for OTLP ingestion over GRPC.
type MonolithicIngestionOTLPProtocolsGRPCSpec struct {
	// Enabled defines if OTLP over gRPC is enabled
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// TLS defines the TLS configuration for OTLP/gRPC ingestion
	//
	// +kubebuilder:validation:Optional
	TLS *TLSSpec `json:"tls,omitempty"`
}

// MonolithicIngestionOTLPProtocolsHTTPSpec defines the settings for OTLP ingestion over HTTP.
type MonolithicIngestionOTLPProtocolsHTTPSpec struct {
	// Enabled defines if OTLP over HTTP is enabled
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// TLS defines the TLS configuration for OTLP/HTTP ingestion
	//
	// +kubebuilder:validation:Optional
	TLS *TLSSpec `json:"tls,omitempty"`
}

// MonolithicJaegerUISpec defines the settings for the Jaeger UI.
type MonolithicJaegerUISpec struct {
	// Enabled defines if the Jaeger UI should be enabled
	//
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`

	// Ingress defines the ingress configuration for Jaeger UI
	//
	// +kubebuilder:validation:Optional
	Ingress *MonolithicJaegerUIIngressSpec `json:"ingress,omitempty"`

	// Route defines the route configuration for Jaeger UI
	//
	// +kubebuilder:validation:Optional
	Route *MonolithicJaegerUIRouteSpec `json:"route,omitempty"`

	// Resources defines the compute resource requirements of Jaeger UI.
	//
	// +kubebuilder:validation:Optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// MonolithicJaegerUIIngressSpec defines the settings for the Jaeger UI ingress.
type MonolithicJaegerUIIngressSpec struct {
	// Enabled defines if an Ingress object should be created for Jaeger UI
	//
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`

	// Annotations defines the annotations of the Ingress object.
	//
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Host defines the hostname of the Ingress object.
	//
	// +kubebuilder:validation:Optional
	Host string `json:"host,omitempty"`

	// IngressClassName is the name of an IngressClass cluster resource. Ingress
	// controller implementations use this field to know whether they should be
	// serving this Ingress resource.
	//
	// +kubebuilder:validation:Optional
	IngressClassName *string `json:"ingressClassName,omitempty"`
}

// MonolithicJaegerUIRouteSpec defines the settings for the Jaeger UI route.
type MonolithicJaegerUIRouteSpec struct {
	// Enabled defines if a Route object should be created for Jaeger UI
	//
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`

	// Annotations defines the annotations of the Route object.
	//
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Host defines the hostname of the Route object.
	//
	// +kubebuilder:validation:Optional
	Host string `json:"host,omitempty"`

	// Termination specifies the termination type. Default: edge.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=edge
	Termination TLSRouteTerminationType `json:"termination,omitempty"`
}

// TempoMonolithicStatus defines the observed state of TempoMonolithic.
type TempoMonolithicStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Storage",type="string",JSONPath=".spec.storage.traces.backend",description="Storage"

// TempoMonolithic is the Schema for the tempomonolithics API.
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

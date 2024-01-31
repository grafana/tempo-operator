package v1alpha1

import (
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

	// Observability defines observability configuration for the Tempo deployment
	//
	// +kubebuilder:validation:Optional
	Observability *MonolithicObservabilitySpec `json:"observability,omitempty"`

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

	// WAL defines the write-ahead logging (WAL) configuration
	//
	// +kubebuilder:validation:Optional
	WAL *MonolithicTracesStorageWALSpec `json:"wal,omitempty"`

	// PV defines the Persistent Volume configuration
	//
	// +kubebuilder:validation:Optional
	PV *MonolithicTracesStoragePVSpec `json:"pv,omitempty"`
}

// MonolithicTracesStorageBackend defines the backend storage for traces.
//
// +kubebuilder:validation:Enum=memory;pv
type MonolithicTracesStorageBackend string

const (
	// MonolithicTracesStorageBackendMemory defines storing traces in a tmpfs (in-memory filesystem).
	MonolithicTracesStorageBackendMemory MonolithicTracesStorageBackend = "memory"
	// MonolithicTracesStorageBackendPV defines storing traces in a Persistent Volume.
	MonolithicTracesStorageBackendPV MonolithicTracesStorageBackend = "pv"
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

// MonolithicObservabilitySpec defines the observability settings of the Tempo deployment.
type MonolithicObservabilitySpec struct {
	// Metrics defines the metrics configuration of the Tempo deployment
	//
	// +kubebuilder:validation:Optional
	Metrics *MonolithicObservabilityMetricsSpec `json:"metrics,omitempty"`
}

// MonolithicObservabilityMetricsSpec defines the metrics settings of the Tempo deployment.
type MonolithicObservabilityMetricsSpec struct {
	// ServiceMonitors defines the ServiceMonitor configuration
	//
	// +kubebuilder:validation:Optional
	ServiceMonitors *MonolithicObservabilityMetricsServiceMonitorsSpec `json:"serviceMonitors,omitempty"`

	// ServiceMonitors defines the PrometheusRule configuration
	//
	// +kubebuilder:validation:Optional
	PrometheusRules *MonolithicObservabilityMetricsPrometheusRulesSpec `json:"prometheusRules,omitempty"`
}

// MonolithicObservabilityMetricsServiceMonitorsSpec defines the ServiceMonitor settings.
type MonolithicObservabilityMetricsServiceMonitorsSpec struct {
	// Enabled defines if the operator should create ServiceMonitors for this Tempo deployment
	//
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`
}

// MonolithicObservabilityMetricsPrometheusRulesSpec defines the PrometheusRules settings.
type MonolithicObservabilityMetricsPrometheusRulesSpec struct {
	// Enabled defines if the operator should create PrometheusRules for this Tempo deployment
	//
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`
}

// TempoMonolithicStatus defines the observed state of TempoMonolithic.
type TempoMonolithicStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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

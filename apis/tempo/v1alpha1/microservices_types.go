package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MicroservicesSpec defines the desired state of Microservices.
type MicroservicesSpec struct {
	// Images defines the image for each container.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Container Images"
	Images ImagesSpec `json:"images,omitempty"`

	// ServiceAccount defines the service account to use for all tempo components.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Service Account"
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// NOTE: currently this field is not considered.
	// Components defines requirements for a set of tempo components.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tempo Components"
	Components TempoComponentsSpec `json:"template,omitempty"`

	// Resources defines resources configuration.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resources"
	Resources Resources `json:"resources,omitempty"`

	// Storage defines S3 compatible object storage configuration.
	// User is required to create secret and supply it.
	//
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Object Storage"
	Storage ObjectStorageSpec `json:"storage"`

	// NOTE: currently this field is not considered.
	// Retention period defined by dataset.
	// User can specify how long data should be stored.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Retention Period"
	Retention RetentionSpec `json:"retention,omitempty"`

	// StorageClassName for PVCs used by ingester. Defaults to nil (default storage class in the cluster).
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="StorageClassName for PVCs"
	StorageClassName *string `json:"storageClassName,omitempty"`

	// NOTE: currently this field is not considered.
	// LimitSpec is used to limit ingestion and querying rates.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ingestion and Querying Ratelimiting"
	LimitSpec LimitSpec `json:"limits,omitempty"`

	// StorageSize for PVCs used by ingester. Defaults to 10Gi.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Storage size for PVCs"
	StorageSize resource.Quantity `json:"storageSize,omitempty"`

	// SearchSpec control the configuration for the search capabilities.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Search configuration options"
	SearchSpec SearchSpec `json:"search,omitempty"`

	// NOTE: currently this field is not considered.
	// ReplicationFactor is used to define how many component replicas should exist.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Replication Factor"
	ReplicationFactor int `json:"replicationFactor,omitempty"`
}

// MicroservicesStatus defines the observed state of Microservices.
type MicroservicesStatus struct {
	// Version of the managed Tempo instance.
	// +optional
	TempoVersion string `json:"tempoVersion,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="Tempo version",type="string",JSONPath=".status.tempoVersion",description="Tempo Version"

// Microservices is the Schema for the microservices API.
type Microservices struct {
	Status            MicroservicesStatus `json:"status,omitempty"`
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MicroservicesSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// MicroservicesList contains a list of Microservices.
type MicroservicesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Microservices `json:"items"`
}

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
}

// Resources defines resources configuration.
type Resources struct {
	// The total amount of resources for Tempo instance.
	// The operator autonomously splits resources between deployed Tempo components.
	// Only limits are supported, the operator calculates requests automatically.
	// See http://github.com/grafana/tempo/issues/1540.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Requirements"
	Total *corev1.ResourceRequirements `json:"total,omitempty"`
}

// SearchSpec specified the global search parameters.
type SearchSpec struct {
	// Enable tempo search feature, default to true
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Search enabled"
	Enabled *bool `json:"enabled,omitempty"`
	// Limit used for search requests if none is set by the caller (default: 20)
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Limit used for search requests if none is set by the caller, this limit the number of traces returned by the query"
	DefaultResultLimit *int `json:"defaultResultLimit,omitempty"`
	// The maximum allowed time range for a search, default: 0s which means unlimited.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Max search time range allowed"
	MaxDuration metav1.Duration `json:"maxDuration,omitempty"`
	// The maximum allowed value of the limit parameter on search requests. If the search request limit parameter
	// exceeds the value configured here it will be set to the value configured here.
	// The default value of 0 disables this limit.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="The maximum allowed value of the limit parameter on search requests, this determine the max number of traces allowed to be returned"
	MaxResultLimit int `json:"maxResultLimit,omitempty"`
}

// ObjectStorageSpec defines the requirements to access the object
// storage bucket to persist traces by the ingester component.
type ObjectStorageSpec struct {
	// TLS configuration for reaching the object storage endpoint.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="TLS Config"
	TLS *ObjectStorageTLSSpec `json:"tls,omitempty"`

	// Secret for object storage authentication.
	// Name of a secret in the same namespace as the tempo Microservices custom resource.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Object Storage Secret"
	Secret string `json:"secret"`
	// Don't forget to update storageSecretField in microservices_controller.go if this field name changes.
}

// ObjectStorageTLSSpec is the TLS configuration for reaching the object storage endpoint.
type ObjectStorageTLSSpec struct {
	// CA is the name of a ConfigMap containing a CA certificate.
	// It needs to be in the same namespace as the LokiStack custom resource.
	//
	// +optional
	// +kubebuilder:validation:optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:io.kubernetes:ConfigMap",displayName="CA ConfigMap Name"
	CA string `json:"caName,omitempty"`
}

// TempoComponentsSpec defines the template of all requirements to configure
// scheduling of all Tempo components to be deployed.
type TempoComponentsSpec struct {
	// Distributor defines the distributor component spec.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Distributor pods"
	Distributor *TempoComponentSpec `json:"distributor,omitempty"`

	// Ingester defines the ingester component spec.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ingester pods"
	Ingester *TempoComponentSpec `json:"ingester,omitempty"`

	// Compactor defines the lokistack compactor component spec.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Compactor pods"
	Compactor *TempoComponentSpec `json:"compactor,omitempty"`

	// Querier defines the querier component spec.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Querier pods"
	Querier *TempoComponentSpec `json:"querier,omitempty"`

	// TempoQueryFrontendSpec defines the query frontend spec.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Query Frontend pods"
	QueryFrontend *TempoQueryFrontendSpec `json:"queryFrontend,omitempty"`
}

// TempoComponentSpec defines specific schedule settings for tempo components.
type TempoComponentSpec struct {
	// Replicas represents the number of replicas to create for this component.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Component Replicas"
	Replicas *int32 `json:"replicas,omitempty"`

	// NodeSelector is the simplest recommended form of node selection constraint.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Selector"
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations defines component specific pod tolerations.
	//
	// +optional
	// +listType=atomic
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tolerations"
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// TempoQueryFrontendSpec extends TempoComponentSpec with frontend specific parameters.
type TempoQueryFrontendSpec struct {
	// TempoComponentSpec is embedded to extend this definition with further options.
	//
	// Currently there is no way to inline this field.
	// See: https://github.com/golang/go/issues/6213
	//
	// +optional
	// +kubebuilder:validation:Optional
	TempoComponentSpec `json:"component,omitempty"`

	// JaegerQuerySpec defines Jaeger Query spefic options.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Jaeger Query Settings"
	JaegerQuery JaegerQuerySpec `json:"jaegerQuery"`
}

// JaegerQuerySpec defines Jaeger Query options.
type JaegerQuerySpec struct {
	// Enabled is used to define if Jaeger Query component should be created.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Jaeger Query Enabled"
	Enabled bool `json:"enabled"`
}

// LimitSpec defines Global and PerTenant rate limits.
type LimitSpec struct {
	// PerTenant is used to define rate limits per tenant.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tenant Limits"
	PerTenant map[string]RateLimitSpec `json:"perTenant,omitempty"`

	// Global is used to define global rate limits.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Global Limit"
	Global RateLimitSpec `json:"global"`
}

// RateLimitSpec defines rate limits for Ingestion and Query components.
type RateLimitSpec struct {
	// Ingestion is used to define ingestion rate limits.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ingestion Limit"
	Ingestion IngestionLimitSpec `json:"ingestion"`

	// Query is used to define query rate limits.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Query Limit"
	Query QueryLimit `json:"query"`
}

// IngestionLimitSpec defines the limits applied at the ingestion path.
type IngestionLimitSpec struct {
	// IngestionBurstSizeBytes defines the burst size (bytes) used in ingestion.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:com.tectonic.ui:number",displayName="Ingestion Burst Size in Bytes"
	IngestionBurstSizeBytes *int `json:"ingestionBurstSizeBytes,omitempty"`

	// IngestionRateLimitBytes defines the Per-user ingestion rate limit (bytes) used in ingestion.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:com.tectonic.ui:number",displayName="Ingestion Rate Limit in Bytes"
	IngestionRateLimitBytes *int `json:"ingestionRateLimitBytes,omitempty"`

	// MaxBytesPerTrace defines the maximum number of bytes of an acceptable trace.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:com.tectonic.ui:number",displayName="Max Bytes per Trace"
	MaxBytesPerTrace *int `json:"maxBytesPerTrace,omitempty"`

	// MaxTracesPerUser defines the maximum number of traces a user can send.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:com.tectonic.ui:number",displayName="Max Traces per User"
	MaxTracesPerUser *int `json:"maxTracesPerUser,omitempty"`
}

// QueryLimit defines query limits.
type QueryLimit struct {
	// MaxBytesPerTagValues defines the maximum size in bytes of a tag-values query.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:com.tectonic.ui:number",displayName="Max Tags per User"
	MaxBytesPerTagValues *int `json:"maxBytesPerTagValues,omitempty"`
	// MaxSearchBytesPerTrace defines the maximum size of search data for a single
	// trace in bytes.
	// default: `0` to disable.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:com.tectonic.ui:number",displayName="Max Traces per User"
	MaxSearchBytesPerTrace *int `json:"maxSearchBytesPerTrace"`
}

// RetentionSpec defines global and per tenant retention configurations.
type RetentionSpec struct {
	// PerTenant is used to configure retention per tenant.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="PerTenant Retention"
	PerTenant map[string]RetentionConfig `json:"perTenant,omitempty"`
	// Global is used to configure global retention.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Global Retention"
	Global RetentionConfig `json:"global"`
}

// RetentionConfig defines how long data should be provided.
type RetentionConfig struct {
	// Traces defines retention period. Supported parameter suffixes are “s”, “m” and “h”.
	// example: 336h
	// default: value is 48h.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:com.tectonic.ui:text",displayName="Trace Retention Period"
	Traces metav1.Duration `json:"traces"`
}

func init() {
	SchemeBuilder.Register(&Microservices{}, &MicroservicesList{})
}

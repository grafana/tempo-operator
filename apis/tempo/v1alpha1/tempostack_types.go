package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/apis/config/v1alpha1"
)

// ManagementStateType defines the type for CR management states.
//
// +kubebuilder:validation:Enum=Managed;Unmanaged
type ManagementStateType string

const (
	// ManagementStateManaged when the TempoStack custom resource should be
	// reconciled by the operator.
	ManagementStateManaged ManagementStateType = "Managed"

	// ManagementStateUnmanaged when the TempoStack custom resource should not be
	// reconciled by the operator.
	ManagementStateUnmanaged ManagementStateType = "Unmanaged"
)

// TempoStackSpec defines the desired state of TempoStack.
type TempoStackSpec struct {
	// ManagementState defines if the CR should be managed by the operator or not.
	// Default is managed.
	//
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:default:=Managed
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:select:Managed","urn:alm:descriptor:com.tectonic.ui:select:Unmanaged"},displayName="Management State"
	ManagementState ManagementStateType `json:"managementState,omitempty"`

	// LimitSpec is used to limit ingestion and querying rates.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ingestion and Querying Ratelimiting"
	LimitSpec LimitSpec `json:"limits,omitempty"`

	// StorageClassName for PVCs used by ingester. Defaults to nil (default storage class in the cluster).
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="StorageClassName for PVCs"
	StorageClassName *string `json:"storageClassName,omitempty"`

	// Resources defines resources configuration.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resources"
	Resources Resources `json:"resources,omitempty"`

	// StorageSize for PVCs used by ingester. Defaults to 10Gi.
	//
	// +optional
	// +kubebuilder:default="10Gi"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Storage size for PVCs"
	StorageSize resource.Quantity `json:"storageSize,omitempty"`

	// Images defines the image for each container.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Container Images"
	Images v1alpha1.ImagesSpec `json:"images,omitempty"`

	// Storage defines the spec for the object storage endpoint to store traces.
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

	// ServiceAccount defines the service account to use for all tempo components.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Service Account"
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// SearchSpec control the configuration for the search capabilities.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Search configuration options"
	SearchSpec SearchSpec `json:"search,omitempty"`

	// HashRing defines the spec for the distributed hash ring configuration.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:com.tectonic.ui:advanced",displayName="Hash Ring"
	HashRing HashRingSpec `json:"hashRing,omitempty"`

	// Template defines requirements for a set of tempo components.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tempo Component Templates"
	Template TempoTemplateSpec `json:"template,omitempty"`

	// ReplicationFactor is used to define how many component replicas should exist.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Replication Factor"
	ReplicationFactor int `json:"replicationFactor,omitempty"`

	// Tenants defines the per-tenant authentication and authorization spec.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tenants Configuration"
	Tenants *TenantsSpec `json:"tenants,omitempty"`

	// ObservabilitySpec defines how telemetry data gets handled.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Observability"
	Observability ObservabilitySpec `json:"observability,omitempty"`

	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Extra Configurations"
	ExtraConfig *ExtraConfigSpec `json:"extraConfig,omitempty"`
}

// ObservabilitySpec defines how telemetry data gets handled.
type ObservabilitySpec struct {
	// Metrics defines the metrics configuration for operands.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Metrics Config"
	Metrics MetricsConfigSpec `json:"metrics,omitempty"`

	// Tracing defines a config for operands.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tracing Config"
	Tracing TracingConfigSpec `json:"tracing,omitempty"`

	// Grafana defines the Grafana configuration for operands.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Grafana Config"
	Grafana GrafanaConfigSpec `json:"grafana,omitempty"`
}

// MetricsConfigSpec defines a metrics config.
type MetricsConfigSpec struct {
	// CreateServiceMonitors specifies if ServiceMonitors should be created for Tempo components.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Create ServiceMonitors for Tempo components"
	CreateServiceMonitors bool `json:"createServiceMonitors,omitempty"`

	// CreatePrometheusRules specifies if Prometheus rules for alerts should be created for Tempo components.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Create PrometheusRules for Tempo components"
	CreatePrometheusRules bool `json:"createPrometheusRules,omitempty"`
}

// TracingConfigSpec defines a tracing config including endpoints and sampling.
type TracingConfigSpec struct {
	// SamplingFraction defines the sampling ratio. Valid values are 0 to 1.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Sampling Fraction"
	SamplingFraction string `json:"sampling_fraction,omitempty"`

	// JaegerAgentEndpoint defines the jaeger endpoint data gets send to.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="localhost:6831"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Jaeger-Agent-Endpoint"
	JaegerAgentEndpoint string `json:"jaeger_agent_endpoint,omitempty"`
}

// GrafanaConfigSpec defines configuration for Grafana.
type GrafanaConfigSpec struct {
	// CreateDatasource specifies if a Grafana Datasource should be created for Tempo.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Create Datasource for Tempo"
	CreateDatasource bool `json:"createDatasource,omitempty"`

	// InstanceSelector specifies the Grafana instance where the datasource should be created.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Create CreateDatasource for Tempo"
	InstanceSelector metav1.LabelSelector `json:"instanceSelector,omitempty"`
}

// ComponentStatus defines the status of each component.
type ComponentStatus struct {
	// Compactor is a map to the pod status of the compactor pod.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podStatuses",displayName="Compactor",order=5
	Compactor PodStatusMap `json:"compactor"`

	// Distributor is a map to the per pod status of the distributor deployment
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podStatuses",displayName="Distributor",order=1
	Distributor PodStatusMap `json:"distributor"`

	// Ingester is a map to the per pod status of the ingester statefulset
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podStatuses",displayName="Ingester",order=2
	Ingester PodStatusMap `json:"ingester"`

	// Querier is a map to the per pod status of the querier deployment
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podStatuses",displayName="Querier",order=3
	Querier PodStatusMap `json:"querier"`

	// QueryFrontend is a map to the per pod status of the query frontend deployment
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podStatuses",displayName="Query Frontend",order=4
	QueryFrontend PodStatusMap `json:"queryFrontend"`

	// Gateway is a map to the per pod status of the query frontend deployment
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podStatuses",displayName="Gateway",order=6
	Gateway PodStatusMap `json:"gateway"`
}

// TempoStackStatus defines the observed state of TempoStack.
type TempoStackStatus struct {
	// Version of the Tempo Operator.
	// +optional
	OperatorVersion string `json:"operatorVersion,omitempty"`

	// Version of the managed Tempo instance.
	// +optional
	TempoVersion string `json:"tempoVersion,omitempty"`

	// DEPRECATED. Version of the Tempo Query component used.
	// +optional
	TempoQueryVersion string `json:"tempoQueryVersion,omitempty"`

	// Components provides summary of all Tempo pod status grouped
	// per component.
	//
	// +optional
	// +kubebuilder:validation:Optional
	Components ComponentStatus `json:"components,omitempty"`

	// Conditions of the Tempo deployment health.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:io.kubernetes.conditions"
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ConditionStatus defines the status of a condition (e.g. ready, failed, pending or configuration error).
type ConditionStatus string

const (
	// ConditionReady defines that all components are ready.
	ConditionReady ConditionStatus = "Ready"
	// ConditionFailed defines that one or more components are in a failed state.
	ConditionFailed ConditionStatus = "Failed"
	// ConditionPending defines that one or more components are in a pending state.
	ConditionPending ConditionStatus = "Pending"
	// ConditionConfigurationError defines that there is a configuration error.
	ConditionConfigurationError ConditionStatus = "ConfigurationError"
)

// AllStatusConditions lists all possible status conditions.
var AllStatusConditions = []ConditionStatus{ConditionReady, ConditionFailed, ConditionPending, ConditionConfigurationError}

// ConditionReason defines possible reasons for each condition.
type ConditionReason string

const (
	// ReasonReady defines a healthy tempo instance.
	ReasonReady ConditionReason = "Ready"
	// ReasonInvalidStorageConfig defines that the object storage configuration is invalid (missing or incomplete storage secret).
	ReasonInvalidStorageConfig ConditionReason = "InvalidStorageConfig"
	// ReasonFailedComponents when all/some Tempo components fail to roll out.
	ReasonFailedComponents ConditionReason = "FailedComponents"
	// ReasonPendingComponents when all/some Tempo components pending dependencies.
	ReasonPendingComponents ConditionReason = "PendingComponents"
	// ReasonCouldNotGetOpenShiftBaseDomain when operator cannot get OpenShift base domain, that is used for OAuth redirect URL.
	ReasonCouldNotGetOpenShiftBaseDomain ConditionReason = "CouldNotGetOpenShiftBaseDomain"
	// ReasonCouldNotGetOpenShiftTLSPolicy when operator cannot get OpenShift TLS security cluster policy.
	ReasonCouldNotGetOpenShiftTLSPolicy ConditionReason = "CouldNotGetOpenShiftTLSPolicy"
	// ReasonMissingGatewayTenantSecret when operator cannot get Secret containing sensitive Gateway information.
	ReasonMissingGatewayTenantSecret ConditionReason = "ReasonMissingGatewayTenantSecret"
	// ReasonInvalidTenantsConfiguration when the tenant configuration provided is invalid.
	ReasonInvalidTenantsConfiguration ConditionReason = "InvalidTenantsConfiguration"
	// ReasonFailedReconciliation when the operator failed to reconcile.
	ReasonFailedReconciliation ConditionReason = "FailedReconciliation"
)

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

// ObjectStorageSecretType defines the type of storage which can be used with the Tempo cluster.
//
// +kubebuilder:validation:Enum=azure;gcs;s3
type ObjectStorageSecretType string

const (
	// ObjectStorageSecretAzure when using Azure Storage for Tempo storage.
	ObjectStorageSecretAzure ObjectStorageSecretType = "azure"

	// ObjectStorageSecretGCS when using Google Cloud Storage for Tempo storage.
	ObjectStorageSecretGCS ObjectStorageSecretType = "gcs"

	// ObjectStorageSecretS3 when using S3 for Tempo storage.
	ObjectStorageSecretS3 ObjectStorageSecretType = "s3"
)

// ObjectStorageSecretSpec is a secret reference containing name only, no namespace.
type ObjectStorageSecretSpec struct {
	// Type of object storage that should be used
	//
	// +required
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:select:azure","urn:alm:descriptor:com.tectonic.ui:select:gcs","urn:alm:descriptor:com.tectonic.ui:select:s3"},displayName="Object Storage Secret Type"
	Type ObjectStorageSecretType `json:"type"`

	// Name of a secret in the namespace configured for object storage secrets.
	//
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:io.kubernetes:Secret",displayName="Object Storage Secret Name"
	Name string `json:"name"`
}

// ObjectStorageSpec defines the requirements to access the object
// storage bucket to persist traces by the ingester component.
type ObjectStorageSpec struct {
	// TLS configuration for reaching the object storage endpoint.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="TLS Config"
	TLS TLSSpec `json:"tls,omitempty"`

	// Secret for object storage authentication.
	// Name of a secret in the same namespace as the TempoStack custom resource.
	//
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Object Storage Secret"
	Secret ObjectStorageSecretSpec `json:"secret"`
	// Don't forget to update storageSecretField in tempostack_controller.go if this field name changes.
}

// MemberListSpec defines the configuration for the memberlist based hash ring.
type MemberListSpec struct {
	// EnableIPv6 enables IPv6 support for the memberlist based hash ring.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch",displayName="Enable IPv6"
	EnableIPv6 *bool `json:"enableIPv6,omitempty"`
}

// HashRingSpec defines the hash ring configuration.
type HashRingSpec struct {
	// MemberList configuration spec
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Memberlist Config"
	MemberList MemberListSpec `json:"memberlist,omitempty"`
}

// TempoTemplateSpec defines the template of all requirements to configure
// scheduling of all Tempo components to be deployed.
type TempoTemplateSpec struct {
	// Distributor defines the distributor component spec.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Distributor pods"
	Distributor TempoDistributorSpec `json:"distributor,omitempty"`

	// Ingester defines the ingester component spec.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ingester pods"
	Ingester TempoComponentSpec `json:"ingester,omitempty"`

	// Compactor defines the tempo compactor component spec.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Compactor pods"
	Compactor TempoComponentSpec `json:"compactor,omitempty"`

	// Querier defines the querier component spec.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Querier pods"
	Querier TempoComponentSpec `json:"querier,omitempty"`

	// TempoQueryFrontendSpec defines the query frontend spec.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Query Frontend pods"
	QueryFrontend TempoQueryFrontendSpec `json:"queryFrontend,omitempty"`

	// Gateway defines the tempo gateway spec.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Gateway pods"
	Gateway TempoGatewaySpec `json:"gateway,omitempty"`
}

// TempoDistributorSpec defines the template of all requirements to configure
// scheduling of Tempo distributor component to be deployed.
type TempoDistributorSpec struct {
	// TempoComponentSpec is embedded to extend this definition with further options.
	//
	// Currently, there is no way to inline this field.
	// See: https://github.com/golang/go/issues/6213
	//
	// +optional
	// +kubebuilder:validation:Optional
	TempoComponentSpec `json:"component,omitempty"`

	// TLS defines TLS configuration for distributor receivers
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="TLS"
	TLS TLSSpec `json:"tls,omitempty"`
}

// TempoComponentSpec defines specific schedule settings for tempo components.
type TempoComponentSpec struct {
	// Replicas defines the number of replicas to be created for this component.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Component Replicas"
	Replicas *int32 `json:"replicas,omitempty"`

	// NodeSelector defines the simple form of the node-selection constraint.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Selector"
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations defines component-specific pod tolerations.
	//
	// +optional
	// +listType=atomic
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tolerations"
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Resources defines resources for this component, this will override the calculated resources derived from total
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resources"
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// TempoGatewaySpec extends TempoComponentSpec with gateway parameters.
type TempoGatewaySpec struct {
	// TempoComponentSpec is embedded to extend this definition with further options.
	//
	// Currently there is no way to inline this field.
	// See: https://github.com/golang/go/issues/6213
	//
	// +optional
	// +kubebuilder:validation:Optional
	TempoComponentSpec `json:"component,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enabled",xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	Enabled bool `json:"enabled"`

	// Ingress defines gateway Ingress options.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Jaeger gateway Ingress Settings"
	Ingress IngressSpec `json:"ingress,omitempty"`
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

	// JaegerQuery defines options specific to the Jaeger Query component.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Jaeger Query Settings"
	JaegerQuery JaegerQuerySpec `json:"jaegerQuery"`
}

// JaegerQuerySpec defines Jaeger Query options.
type JaegerQuerySpec struct {
	// Enabled defines if the Jaeger Query component should be created.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable Jaeger Query UI",xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	Enabled bool `json:"enabled"`

	// Ingress defines the options for the Jaeger Query ingress.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Jaeger Query UI Ingress Settings"
	Ingress IngressSpec `json:"ingress,omitempty"`

	// MonitorTab defines the monitor tab configuration.
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Jaeger Query UI Monitor Tab Settings"
	MonitorTab JaegerQueryMonitor `json:"monitorTab"`
}

// JaegerQueryMonitor defines configuration for the service monitoring tab in the Jaeger console.
// The monitoring tab uses Prometheus to query span RED metrics.
// This feature requires running OpenTelemetry collector with spanmetricsconnector -
// https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/connector/spanmetricsconnector
// which derives span RED metrics from spans and exports the metrics to Prometheus.
type JaegerQueryMonitor struct {
	// Enabled enables the monitor tab in the Jaeger console.
	// The PrometheusEndpoint must be configured to enable this feature.
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enabled",xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	Enabled bool `json:"enabled"`

	// PrometheusEndpoint defines the endpoint to the Prometheus instance that contains the span rate, error, and duration (RED) metrics.
	// For instance on OpenShift this is set to https://thanos-querier.openshift-monitoring.svc.cluster.local:9091
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Prometheus endpoint"
	PrometheusEndpoint string `json:"prometheusEndpoint"`
}

// IngressSpec defines Jaeger Query Ingress options.
type IngressSpec struct {
	// Type defines the type of Ingress for the Jaeger Query UI.
	// Currently ingress, route and none are supported.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Type"
	Type IngressType `json:"type,omitempty"`

	// Annotations defines the annotations of the Ingress object.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Annotations"
	Annotations map[string]string `json:"annotations,omitempty"`

	// Host defines the hostname of the Ingress object.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Host"
	Host string `json:"host,omitempty"`

	// IngressClassName defines the name of an IngressClass cluster resource.
	// Defines which ingress controller serves this ingress resource.
	// +optional
	IngressClassName *string `json:"ingressClassName,omitempty"`

	// Route defines the options for the OpenShift route.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Route Configuration"
	Route RouteSpec `json:"route,omitempty"`
}

// RouteSpec defines OpenShift Route specific options.
type RouteSpec struct {
	// Termination defines the termination type.
	// The default is "edge".
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="TLS Termination Policy"
	Termination TLSRouteTerminationType `json:"termination,omitempty"`
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

	// DEPRECATED. MaxSearchBytesPerTrace defines the maximum size of search data for a single
	// trace in bytes.
	// default: `0` to disable.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:com.tectonic.ui:number",displayName="Max Traces per User"
	MaxSearchBytesPerTrace *int `json:"maxSearchBytesPerTrace,omitempty"`

	// MaxSearchDuration defines the maximum allowed time range for a search.
	// If this value is not set, then spec.search.maxDuration is used.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Max Search Duration per User"
	MaxSearchDuration metav1.Duration `json:"maxSearchDuration"`
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
	// Traces defines retention period. Supported parameter suffixes are "s", "m" and "h".
	// example: 336h
	// default: value is 48h.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:com.tectonic.ui:text",displayName="Trace Retention Period"
	Traces metav1.Duration `json:"traces"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="Tempo Version",type="string",JSONPath=".status.tempoVersion",description="Tempo Version"
//+kubebuilder:printcolumn:name="Management",type="string",JSONPath=".spec.managementState",description="Management State"

// TempoStack manages a Tempo deployment in microservices mode.
//
// +operator-sdk:csv:customresourcedefinitions:displayName="TempoStack",resources={{ConfigMap,v1},{ServiceAccount,v1},{Service,v1},{Secret,v1},{StatefulSet,v1},{Deployment,v1},{Ingress,v1},{Route,v1}}
// +kubebuilder:resource:shortName=tempo;tempos
type TempoStack struct {
	Status            TempoStackStatus `json:"status,omitempty"`
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TempoStackSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// TempoStackList contains a list of TempoStack.
type TempoStackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TempoStack `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TempoStack{}, &TempoStackList{})
}

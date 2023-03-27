package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/os-observability/tempo-operator/apis/config/v1alpha1"
)

// TempoStackSpec defines the desired state of TempoStack.
type TempoStackSpec struct {
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

	// Template defines requirements for a set of tempo components.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tempo Component Templates"
	Template TempoTemplateSpec `json:"template,omitempty"`

	// NOTE: currently this field is not considered.
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
}

// ObservabilitySpec defines how telemetry data gets handled.
type ObservabilitySpec struct {
	// Tracing defines a config for operands.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tracing Config"
	Tracing TracingConfigSpec `json:"tracing,omitempty"`
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

// PodStatusMap defines the type for mapping pod status to pod name.
type PodStatusMap map[corev1.PodPhase][]string

// ComponentStatus defines the status of each component.
type ComponentStatus struct {
	// Compactor is a map to the pod status of the compactor pod.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podStatuses",displayName="Compactor",order=5
	Compactor PodStatusMap `json:"compactor,omitempty"`

	// Distributor is a map to the per pod status of the distributor deployment
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podStatuses",displayName="Distributor",order=1
	Distributor PodStatusMap `json:"distributor,omitempty"`

	// Ingester is a map to the per pod status of the ingester statefulset
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podStatuses",displayName="Ingester",order=2
	Ingester PodStatusMap `json:"ingester,omitempty"`

	// Querier is a map to the per pod status of the querier deployment
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podStatuses",displayName="Querier",order=3
	Querier PodStatusMap `json:"querier,omitempty"`

	// QueryFrontend is a map to the per pod status of the query frontend deployment
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podStatuses",displayName="Query Frontend",order=4
	QueryFrontend PodStatusMap `json:"queryFrontend,omitempty"`

	// Gateway is a map to the per pod status of the query frontend deployment
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podStatuses",displayName="Query Frontend",order=4
	Gateway PodStatusMap `json:"gateway,omitempty"`
}

// TempoStackStatus defines the observed state of TempoStack.
type TempoStackStatus struct {
	// Version of the managed Tempo instance.
	// +optional
	TempoVersion string `json:"tempoVersion,omitempty"`
	// Version of the Tempo Query component used.
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

// ModeType is the authentication/authorization mode in which Tempo Gateway
// will be configured.
//
// +kubebuilder:validation:Enum=static;openshift
type ModeType string

const (
	// Static mode asserts the Authorization Spec's Roles and RoleBindings
	// using an in-process OpenPolicyAgent Rego authorizer.
	Static ModeType = "static"
	// OpenShift mode uses TokenReview API for authentication and subject access review for authorization.
	OpenShift ModeType = "openshift"
)

// TenantsSpec defines the mode, authentication and authorization
// configuration of the tempo gateway component.
type TenantsSpec struct {
	// Mode defines the multitenancy mode.
	//
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:default:=static
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:select:static","urn:alm:descriptor:com.tectonic.ui:select:dynamic"},displayName="Mode"
	Mode ModeType `json:"mode"`

	// Authentication defines the tempo-gateway component authentication configuration spec per tenant.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Authentication"
	Authentication []AuthenticationSpec `json:"authentication,omitempty"`
	// Authorization defines the tempo-gateway component authorization configuration spec per tenant.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Authorization"
	Authorization *AuthorizationSpec `json:"authorization,omitempty"`
}

// ConditionStatus defines the status of a condition (e.g. ready or degraded).
type ConditionStatus string

const (
	// ConditionReady defines that all components are ready.
	ConditionReady ConditionStatus = "Ready"
	// ConditionDegraded defines that one or more components are in a degraded state.
	ConditionDegraded ConditionStatus = "Degraded"
	// ConditionFailed defines that one or more components are in a failed state.
	ConditionFailed ConditionStatus = "Failed"
	// ConditionPending defines that one or more components are in a degraded state.
	ConditionPending ConditionStatus = "Pending"
)

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
)

// PermissionType is a Tempo Gateway RBAC permission.
//
// +kubebuilder:validation:Enum=read;write
type PermissionType string

const (
	// Write gives access to write data to a tenant.
	Write PermissionType = "write"
	// Read gives access to read data from a tenant.
	Read PermissionType = "read"
)

// SubjectKind is a kind of Tempo Gateway RBAC subject.
//
// +kubebuilder:validation:Enum=user;group
type SubjectKind string

const (
	// User represents a subject that is a user.
	User SubjectKind = "user"
	// Group represents a subject that is a group.
	Group SubjectKind = "group"
)

// Subject represents a subject that has been bound to a role.
type Subject struct {
	Name string      `json:"name"`
	Kind SubjectKind `json:"kind"`
}

// RoleBindingsSpec binds a set of roles to a set of subjects.
type RoleBindingsSpec struct {
	Name     string    `json:"name"`
	Subjects []Subject `json:"subjects"`
	Roles    []string  `json:"roles"`
}

// AuthorizationSpec defines the opa, role bindings and roles
// configuration per tenant for tempo Gateway component.
type AuthorizationSpec struct {
	// Roles defines a set of permissions to interact with a tenant.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Static Roles"
	Roles []RoleSpec `json:"roles"`
	// RoleBindings defines configuration to bind a set of roles to a set of subjects.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Static Role Bindings"
	RoleBindings []RoleBindingsSpec `json:"roleBindings"`
}

// RoleSpec describes a set of permissions to interact with a tenant.
type RoleSpec struct {
	Name        string           `json:"name"`
	Resources   []string         `json:"resources"`
	Tenants     []string         `json:"tenants"`
	Permissions []PermissionType `json:"permissions"`
}

// TenantSecretSpec is a secret reference containing name only
// for a secret living in the same namespace as the (Tempo) TempoStack custom resource.
type TenantSecretSpec struct {
	// Name of a secret in the namespace configured for tenant secrets.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:io.kubernetes:Secret",displayName="Tenant Secret Name"
	Name string `json:"name"`
}

// OIDCSpec defines the oidc configuration spec for Tempo Gateway component.
type OIDCSpec struct {
	// Secret defines the spec for the clientID, clientSecret and issuerCAPath for tenant's authentication.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tenant Secret"
	Secret *TenantSecretSpec `json:"secret"`
	// IssuerURL defines the URL for issuer.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Issuer URL"
	IssuerURL string `json:"issuerURL"`
	// RedirectURL defines the URL for redirect.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Redirect URL"
	RedirectURL string `json:"redirectURL,omitempty"`
	// Group claim field from ID Token
	//
	// +optional
	// +kubebuilder:validation:Optional
	GroupClaim string `json:"groupClaim,omitempty"`
	// User claim field from ID Token
	//
	// +optional
	// +kubebuilder:validation:Optional
	UsernameClaim string `json:"usernameClaim,omitempty"`
}

// AuthenticationSpec defines the oidc configuration per tenant for tempo Gateway component.
type AuthenticationSpec struct {
	// TenantName defines the name of the tenant.
	//
	// +required
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tenant Name"
	TenantName string `json:"tenantName"`
	// TenantID defines the id of the tenant.
	//
	// +required
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tenant ID"
	TenantID string `json:"tenantId"`
	// OIDC defines the spec for the OIDC tenant's authentication.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OIDC Configuration"
	OIDC *OIDCSpec `json:"oidc,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="Tempo version",type="string",JSONPath=".status.tempoVersion",description="Tempo Version"

// TempoStack is the Schema for the tempostacks API.
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
	TLS *ObjectStorageTLSSpec `json:"tls,omitempty"`

	// Secret for object storage authentication.
	// Name of a secret in the same namespace as the tempo TempoStack custom resource.
	//
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Object Storage Secret"
	Secret ObjectStorageSecretSpec `json:"secret"`
	// Don't forget to update storageSecretField in tempostack_controller.go if this field name changes.
}

// ObjectStorageTLSSpec is the TLS configuration for reaching the object storage endpoint.
type ObjectStorageTLSSpec struct {
	// CA is the name of a ConfigMap containing a CA certificate.
	// It needs to be in the same namespace as the Tempo custom resource.
	//
	// +optional
	// +kubebuilder:validation:optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:io.kubernetes:ConfigMap",displayName="CA ConfigMap Name"
	CA string `json:"caName,omitempty"`
}

// TempoTemplateSpec defines the template of all requirements to configure
// scheduling of all Tempo components to be deployed.
type TempoTemplateSpec struct {
	// Distributor defines the distributor component spec.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Distributor pods"
	Distributor TempoComponentSpec `json:"distributor,omitempty"`

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

	Enabled bool `json:"enabled"`
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

	// JaegerQuerySpec defines Jaeger Query specific options.
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
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable Jaeger Query UI"
	Enabled bool `json:"enabled"`

	// Ingress defines Jaeger Query Ingress options.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Jaeger Query UI Ingress Settings"
	Ingress JaegerQueryIngressSpec `json:"ingress,omitempty"`
}

// JaegerQueryIngressSpec defines Jaeger Query Ingress options.
type JaegerQueryIngressSpec struct {
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

	// IngressClassName is the name of an IngressClass cluster resource. Ingress
	// controller implementations use this field to know whether they should be
	// serving this Ingress resource.
	// +optional
	IngressClassName *string `json:"ingressClassName,omitempty"`

	// Route defines OpenShift Route specific options.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Route Configuration"
	Route JaegerQueryRouteSpec `json:"route,omitempty"`
}

// JaegerQueryRouteSpec defines OpenShift Route specific options.
type JaegerQueryRouteSpec struct {
	// Termination specifies the termination type. By default "edge" is used.
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
	SchemeBuilder.Register(&TempoStack{}, &TempoStackList{})
}

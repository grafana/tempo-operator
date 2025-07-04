package v1alpha1

// ModeType is the authentication/authorization mode in which Tempo Gateway
// will be configured.
//
// +kubebuilder:validation:Enum=static;openshift
type ModeType string

const (
	// ModeStatic mode asserts the Authorization Spec's Roles and RoleBindings
	// using an in-process OpenPolicyAgent Rego authorizer.
	ModeStatic ModeType = "static"
	// ModeOpenShift mode uses TokenReview API for authentication and SelfSubjectAccessReview for authorization.
	ModeOpenShift ModeType = "openshift"
)

// TenantsSpec defines the mode, authentication and authorization
// configuration of the tempo gateway component.
type TenantsSpec struct {
	// Mode defines the multitenancy mode.
	//
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:default:=static
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:select:static","urn:alm:descriptor:com.tectonic.ui:select:openshift"},displayName="Mode"
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

// AuthenticationSpec defines the oidc configuration per tenant for tempo Gateway component.
type AuthenticationSpec struct {
	// TenantName defines a human readable, unique name of the tenant.
	// The value of this field must be specified in the X-Scope-OrgID header and in the resources field of a ClusterRole to identify the tenant.
	//
	// +required
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tenant Name"
	TenantName string `json:"tenantName"`

	// TenantID defines a universally unique identifier of the tenant.
	// Unlike the tenantName, which must be unique at a given time, the tenantId must be unique over the entire lifetime of the Tempo deployment.
	// Tempo uses this ID to prefix objects in the object storage.
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

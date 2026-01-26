package envconfig

import "errors"

// =============================================================================
// Environment Variable Names
// =============================================================================

// Feature gates environment variable.
const (
	// envFeatureGates is a comma-separated list of feature gates to enable/disable.
	// Format: "gate1,gate2,-gate3" where - prefix disables the gate.
	// Example: "openshift.route,httpEncryption,-networkPolicies".
	envFeatureGates = "FEATURE_GATES"
)

// Gate settings (non-boolean configuration values).
const (
	// envOpenShiftBaseDomain sets the OpenShift base domain for route generation.
	envOpenShiftBaseDomain = "OPENSHIFT_BASE_DOMAIN"
	// envTLSProfile sets the TLS security profile (e.g., "Old", "Intermediate", "Modern").
	envTLSProfile = "TLS_PROFILE"
	// envDefaultPodSecurityContext sets the default PodSecurityContext as JSON.
	// Example: '{"fsGroup": 10001, "runAsNonRoot": true}'.
	envDefaultPodSecurityContext = "DEFAULT_POD_SECURITY_CONTEXT"
	// envDistribution sets the distribution type (e.g., "openshift").
	envDistribution = "DISTRIBUTION"
)

// Built-in certificate management settings.
const (
	// envBuiltInCertManagementCAValidity sets the CA certificate validity duration (e.g., "8760h").
	envBuiltInCertManagementCAValidity = "BUILT_IN_CERT_MANAGEMENT_CA_VALIDITY"
	// envBuiltInCertManagementCARefresh sets the CA certificate refresh interval (e.g., "7008h").
	envBuiltInCertManagementCARefresh = "BUILT_IN_CERT_MANAGEMENT_CA_REFRESH"
	// envBuiltInCertManagementCertValidity sets the certificate validity duration (e.g., "2160h").
	envBuiltInCertManagementCertValidity = "BUILT_IN_CERT_MANAGEMENT_CERT_VALIDITY"
	// envBuiltInCertManagementCertRefresh sets the certificate refresh interval (e.g., "1728h").
	envBuiltInCertManagementCertRefresh = "BUILT_IN_CERT_MANAGEMENT_CERT_REFRESH"
)

// Leader election settings.
const (
	// envLeaderElectionEnabled enables or disables leader election ("true" or "false").
	envLeaderElectionEnabled = "LEADER_ELECTION_ENABLED"
	// envLeaderElectionResourceLock sets the resource lock type (e.g., "leases").
	envLeaderElectionResourceLock = "LEADER_ELECTION_RESOURCE_LOCK"
	// envLeaderElectionResourceNamespace sets the namespace for leader election resources.
	envLeaderElectionResourceNamespace = "LEADER_ELECTION_RESOURCE_NAMESPACE"
	// envLeaderElectionResourceName sets the name of the leader election resource.
	envLeaderElectionResourceName = "LEADER_ELECTION_RESOURCE_NAME"
	// envLeaderElectionLeaseDuration sets the leader election lease duration (e.g., "15s").
	envLeaderElectionLeaseDuration = "LEADER_ELECTION_LEASE_DURATION"
	// envLeaderElectionRenewDeadline sets the leader election renew deadline (e.g., "10s").
	envLeaderElectionRenewDeadline = "LEADER_ELECTION_RENEW_DEADLINE"
	// envLeaderElectionRetryPeriod sets the leader election retry period (e.g., "2s").
	envLeaderElectionRetryPeriod = "LEADER_ELECTION_RETRY_PERIOD"
)

// Controller manager settings (metrics, health, webhook).
const (
	// envMetricsBindAddress sets the metrics server bind address (e.g., ":8080").
	envMetricsBindAddress = "METRICS_BIND_ADDRESS"
	// envMetricsSecure enables secure metrics serving ("true" or "false").
	envMetricsSecure = "METRICS_SECURE"
	// envHealthProbeBindAddress sets the health probe bind address (e.g., ":8081").
	envHealthProbeBindAddress = "HEALTH_PROBE_BIND_ADDRESS"
	// envWebhookPort sets the webhook server port (e.g., "9443").
	envWebhookPort = "WEBHOOK_PORT"
)

// =============================================================================
// Feature Gate Names
// =============================================================================

// Feature gate name constants used in the FEATURE_GATES environment variable.
const (
	// OpenShift-specific feature gates.
	featureGateOpenShiftRoute        = "openshift.route"
	featureGateOpenShiftServingCerts = "openshift.servingCertsService"
	featureGateOpenShiftOAuthProxy   = "openshift.oauthProxy"

	// TLS/Encryption feature gates.
	featureGateHTTPEncryption = "httpEncryption"
	featureGateGRPCEncryption = "grpcEncryption"

	// Operator integration feature gates.
	featureGatePrometheusOperator = "prometheusOperator"
	featureGateGrafanaOperator    = "grafanaOperator"

	// Observability feature gates.
	featureGateCreateServiceMonitors = "observability.metrics.createServiceMonitors"
	featureGateCreatePrometheusRules = "observability.metrics.createPrometheusRules"

	// Other feature gates.
	featureGateNetworkPolicies       = "networkPolicies"
	featureGateBuiltInCertManagement = "builtInCertManagement"
)

// =============================================================================
// Errors
// =============================================================================

// Sentinel errors for feature gate parsing.
var errUnknownFeatureGate = errors.New("unknown feature gate")

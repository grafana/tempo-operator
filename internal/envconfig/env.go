package envconfig

import (
	"os"
	"strconv"
	"strings"

	componentbaseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
)

// featureGateSetter is a function that sets a feature gate value in the config.
type featureGateSetter func(cfg *configv1alpha1.ProjectConfig, enabled bool)

// featureGateDef defines a feature gate with its name and setter function.
// This ensures the name and setter are always defined together, preventing them from getting out of sync.
type featureGateDef struct {
	name   string
	setter featureGateSetter
}

// featureGates defines all available feature gates.
// To add a new feature gate, add a single entry here with both its name and setter.
// Format for FEATURE_GATES env var: "gate1,gate2,-gate3" where - prefix disables the gate.
var featureGates = []featureGateDef{
	// OpenShift feature gates.
	{featureGateOpenShiftRoute, func(cfg *configv1alpha1.ProjectConfig, enabled bool) {
		cfg.Gates.OpenShift.OpenShiftRoute = enabled
	}},
	{featureGateOpenShiftServingCerts, func(cfg *configv1alpha1.ProjectConfig, enabled bool) {
		cfg.Gates.OpenShift.ServingCertsService = enabled
	}},
	{featureGateOpenShiftOAuthProxy, func(cfg *configv1alpha1.ProjectConfig, enabled bool) {
		cfg.Gates.OpenShift.OauthProxy.DefaultEnabled = enabled
	}},
	{featureGateOpenShiftNoAuthWarning, func(cfg *configv1alpha1.ProjectConfig, enabled bool) {
		cfg.Gates.OpenShift.NoAuthWarning = enabled
	}},

	// TLS/Encryption feature gates.
	{featureGateHTTPEncryption, func(cfg *configv1alpha1.ProjectConfig, enabled bool) {
		cfg.Gates.HTTPEncryption = enabled
	}},
	{featureGateGRPCEncryption, func(cfg *configv1alpha1.ProjectConfig, enabled bool) {
		cfg.Gates.GRPCEncryption = enabled
	}},

	// Operators & Observability feature gates.
	{featureGatePrometheusOperator, func(cfg *configv1alpha1.ProjectConfig, enabled bool) {
		cfg.Gates.PrometheusOperator = enabled
	}},
	{featureGateGrafanaOperator, func(cfg *configv1alpha1.ProjectConfig, enabled bool) {
		cfg.Gates.GrafanaOperator = enabled
	}},
	{featureGateCreateServiceMonitors, func(cfg *configv1alpha1.ProjectConfig, enabled bool) {
		cfg.Gates.Observability.Metrics.CreateServiceMonitors = enabled
	}},
	{featureGateCreatePrometheusRules, func(cfg *configv1alpha1.ProjectConfig, enabled bool) {
		cfg.Gates.Observability.Metrics.CreatePrometheusRules = enabled
	}},

	// Other feature gates.
	{featureGateNetworkPolicies, func(cfg *configv1alpha1.ProjectConfig, enabled bool) {
		cfg.Gates.NetworkPolicies = enabled
	}},
	{featureGateBuiltInCertManagement, func(cfg *configv1alpha1.ProjectConfig, enabled bool) {
		cfg.Gates.BuiltInCertManagement.Enabled = enabled
	}},
}

// featureGateSetters is a lookup map built from featureGates for efficient access.
var featureGateSetters = func() map[string]featureGateSetter {
	m := make(map[string]featureGateSetter, len(featureGates))
	for _, fg := range featureGates {
		m[fg.name] = fg.setter
	}
	return m
}()

// ApplyEnvVars applies environment variable overrides to the project configuration.
// Environment variables take precedence over config file values.
func ApplyEnvVars(cfg *configv1alpha1.ProjectConfig) {
	// Feature gates
	if val, ok := os.LookupEnv(envFeatureGates); ok {
		applyFeatureGates(cfg, val)
	}

	// Gate settings (non-boolean values and cert management)
	applyGatesEnvVars(cfg)

	// General configuration
	if val, ok := os.LookupEnv(envDistribution); ok {
		cfg.Distribution = val
	}

	// Controller manager settings
	applyLeaderElectionEnvVars(cfg)
	applyControllerManagerEnvVars(cfg)
}

// applyGatesEnvVars applies non-boolean gate settings and cert management env vars.
func applyGatesEnvVars(cfg *configv1alpha1.ProjectConfig) {
	if val, ok := os.LookupEnv(envOpenShiftBaseDomain); ok {
		cfg.Gates.OpenShift.BaseDomain = val
	}
	if val, ok := os.LookupEnv(envTLSProfile); ok {
		cfg.Gates.TLSProfile = val
	}
	if val, ok := os.LookupEnv(envDefaultPodSecurityContext); ok {
		if psc, err := parsePodSecurityContext(val); err == nil {
			cfg.Gates.DefaultPodSecurityContext = psc
		} else {
			setupLog.Error(err, "invalid value for environment variable, ignoring", "env", envDefaultPodSecurityContext, "value", val)
		}
	}

	// BuiltInCertManagement duration fields
	if d, ok := lookupDurationEnv(envBuiltInCertManagementCAValidity); ok {
		cfg.Gates.BuiltInCertManagement.CACertValidity = d
	}
	if d, ok := lookupDurationEnv(envBuiltInCertManagementCARefresh); ok {
		cfg.Gates.BuiltInCertManagement.CACertRefresh = d
	}
	if d, ok := lookupDurationEnv(envBuiltInCertManagementCertValidity); ok {
		cfg.Gates.BuiltInCertManagement.CertValidity = d
	}
	if d, ok := lookupDurationEnv(envBuiltInCertManagementCertRefresh); ok {
		cfg.Gates.BuiltInCertManagement.CertRefresh = d
	}
}

// applyControllerManagerEnvVars applies metrics, health, and webhook env vars.
func applyControllerManagerEnvVars(cfg *configv1alpha1.ProjectConfig) {
	if val, ok := os.LookupEnv(envMetricsBindAddress); ok {
		cfg.Metrics.BindAddress = val
	}
	if b, ok := lookupBoolEnv(envMetricsSecure); ok {
		cfg.Metrics.Secure = b
	}
	if val, ok := os.LookupEnv(envHealthProbeBindAddress); ok {
		cfg.Health.HealthProbeBindAddress = val
	}
	if val, ok := os.LookupEnv(envWebhookPort); ok {
		if port, err := strconv.Atoi(val); err == nil {
			cfg.Webhook.Port = &port
		} else {
			setupLog.Error(err, "invalid value for environment variable, ignoring", "env", envWebhookPort, "value", val)
		}
	}
}

// parseFeatureGate parses a feature gate string and returns the gate name and enabled state.
// Returns empty name for invalid input (empty string or lone "-").
func parseFeatureGate(gate string) (name string, enabled bool) {
	gate = strings.TrimSpace(gate)
	if name, disabled := strings.CutPrefix(gate, "-"); disabled {
		return name, false
	}
	return gate, true
}

// applyFeatureGates parses a comma-separated list of feature gates and applies them to config.
// Format: "gate1,gate2,-gate3" where - prefix disables the gate.
func applyFeatureGates(cfg *configv1alpha1.ProjectConfig, gates string) {
	for gate := range strings.SplitSeq(gates, ",") {
		name, enabled := parseFeatureGate(gate)
		if name == "" {
			continue
		}

		if setter, ok := featureGateSetters[name]; ok {
			setter(cfg, enabled)
		} else {
			setupLog.Error(errUnknownFeatureGate, "unknown feature gate, ignoring", "gate", name)
		}
	}
}

// applyLeaderElectionEnvVars applies leader election environment variable overrides.
func applyLeaderElectionEnvVars(cfg *configv1alpha1.ProjectConfig) {
	// ensureLeaderElection initializes LeaderElection if nil, called only when we have a valid value to set.
	ensureLeaderElection := func() {
		if cfg.LeaderElection == nil {
			cfg.LeaderElection = &componentbaseconfigv1alpha1.LeaderElectionConfiguration{}
		}
	}

	if b, ok := lookupBoolEnv(envLeaderElectionEnabled); ok {
		ensureLeaderElection()
		cfg.LeaderElection.LeaderElect = &b
	}
	if val, ok := os.LookupEnv(envLeaderElectionResourceLock); ok {
		ensureLeaderElection()
		cfg.LeaderElection.ResourceLock = val
	}
	if val, ok := os.LookupEnv(envLeaderElectionResourceNamespace); ok {
		ensureLeaderElection()
		cfg.LeaderElection.ResourceNamespace = val
	}
	if val, ok := os.LookupEnv(envLeaderElectionResourceName); ok {
		ensureLeaderElection()
		cfg.LeaderElection.ResourceName = val
	}
	if d, ok := lookupDurationEnv(envLeaderElectionLeaseDuration); ok {
		ensureLeaderElection()
		cfg.LeaderElection.LeaseDuration = d
	}
	if d, ok := lookupDurationEnv(envLeaderElectionRenewDeadline); ok {
		ensureLeaderElection()
		cfg.LeaderElection.RenewDeadline = d
	}
	if d, ok := lookupDurationEnv(envLeaderElectionRetryPeriod); ok {
		ensureLeaderElection()
		cfg.LeaderElection.RetryPeriod = d
	}
}

package envconfig

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	componentbaseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"
	"k8s.io/utils/ptr"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
)

const (
	testTLSProfileOld           = "Old"
	testDistributionOpenShift   = "openshift"
	testBaseDomain              = "example.com"
	testResourceLockLeases      = "leases"
	testLeaderElectionNamespace = "tempo-operator-system"
	testLeaderElectionName      = "tempo-operator-lock"
	testMetricsBindAddress      = ":8080"
	testHealthProbeBindAddress  = ":8081"
	testWebhookPort             = "9443"
	testLeaseDuration           = "15s"
	testRenewDeadline           = "10s"
	testRetryPeriod             = "2s"
)

// mustParseDuration parses a duration string or panics.
// Used in tests to derive expected values from input constants.
func mustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic("invalid test duration: " + s)
	}
	return d
}

// mustParseInt parses an int string or panics.
// Used in tests to derive expected values from input constants.
func mustParseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic("invalid test int: " + s)
	}
	return i
}

func TestApplyEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		initial  configv1alpha1.ProjectConfig
		expected configv1alpha1.ProjectConfig
	}{
		{
			name:     "no env vars set",
			envVars:  map[string]string{},
			initial:  configv1alpha1.DefaultProjectConfig(),
			expected: configv1alpha1.DefaultProjectConfig(),
		},
		{
			name: "feature gates - single gate enabled",
			envVars: map[string]string{
				envFeatureGates: "openshift.route",
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					OpenShift: configv1alpha1.OpenShiftFeatureGates{
						OpenShiftRoute: true,
					},
				},
			},
		},
		{
			name: "feature gates - multiple gates enabled",
			envVars: map[string]string{
				envFeatureGates: "openshift.route,httpEncryption,grpcEncryption",
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					OpenShift: configv1alpha1.OpenShiftFeatureGates{
						OpenShiftRoute: true,
					},
					HTTPEncryption: true,
					GRPCEncryption: true,
				},
			},
		},
		{
			name: "feature gates - disable gate with minus prefix",
			envVars: map[string]string{
				envFeatureGates: "-networkPolicies",
			},
			initial: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					NetworkPolicies: true,
				},
			},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					NetworkPolicies: false,
				},
			},
		},
		{
			name: "feature gates - mixed enable and disable",
			envVars: map[string]string{
				envFeatureGates: "prometheusOperator,observability.metrics.createServiceMonitors,-networkPolicies",
			},
			initial: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					NetworkPolicies: true,
				},
			},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					PrometheusOperator: true,
					NetworkPolicies:    false,
					Observability: configv1alpha1.ObservabilityFeatureGates{
						Metrics: configv1alpha1.MetricsFeatureGates{
							CreateServiceMonitors: true,
						},
					},
				},
			},
		},
		{
			name: "feature gates - all boolean gates",
			envVars: map[string]string{
				envFeatureGates: "openshift.route,openshift.servingCertsService,openshift.oauthProxy,httpEncryption,grpcEncryption,prometheusOperator,grafanaOperator,observability.metrics.createServiceMonitors,observability.metrics.createPrometheusRules,networkPolicies,builtInCertManagement",
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					OpenShift: configv1alpha1.OpenShiftFeatureGates{
						OpenShiftRoute:      true,
						ServingCertsService: true,
						OauthProxy: configv1alpha1.OauthProxyFeatureGates{
							DefaultEnabled: true,
						},
					},
					HTTPEncryption:     true,
					GRPCEncryption:     true,
					PrometheusOperator: true,
					GrafanaOperator:    true,
					Observability: configv1alpha1.ObservabilityFeatureGates{
						Metrics: configv1alpha1.MetricsFeatureGates{
							CreateServiceMonitors: true,
							CreatePrometheusRules: true,
						},
					},
					NetworkPolicies: true,
					BuiltInCertManagement: configv1alpha1.BuiltInCertManagement{
						Enabled: true,
					},
				},
			},
		},
		{
			name: "feature gates - with spaces",
			envVars: map[string]string{
				envFeatureGates: "openshift.route, httpEncryption, grpcEncryption",
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					OpenShift: configv1alpha1.OpenShiftFeatureGates{
						OpenShiftRoute: true,
					},
					HTTPEncryption: true,
					GRPCEncryption: true,
				},
			},
		},
		{
			name: "feature gates - unknown gate ignored",
			envVars: map[string]string{
				envFeatureGates: "unknownGate,openshift.route",
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					OpenShift: configv1alpha1.OpenShiftFeatureGates{
						OpenShiftRoute: true,
					},
				},
			},
		},
		{
			name: "feature gates - empty string",
			envVars: map[string]string{
				envFeatureGates: "",
			},
			initial:  configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{},
		},
		{
			name: "openshift base domain",
			envVars: map[string]string{
				envOpenShiftBaseDomain: testBaseDomain,
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					OpenShift: configv1alpha1.OpenShiftFeatureGates{
						BaseDomain: testBaseDomain,
					},
				},
			},
		},
		{
			name: "tls profile",
			envVars: map[string]string{
				envTLSProfile: testTLSProfileOld,
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					TLSProfile: testTLSProfileOld,
				},
			},
		},
		{
			name: "default pod security context",
			envVars: map[string]string{
				envDefaultPodSecurityContext: `{"fsGroup": 10001}`,
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					DefaultPodSecurityContext: &corev1.PodSecurityContext{
						FSGroup: ptr.To(int64(10001)),
					},
				},
			},
		},
		{
			name: "default pod security context - invalid json ignored",
			envVars: map[string]string{
				envDefaultPodSecurityContext: `invalid`,
			},
			initial:  configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{}, // unchanged due to invalid JSON
		},
		{
			name: "builtin cert management durations",
			envVars: map[string]string{
				envBuiltInCertManagementCAValidity:   "8760h",
				envBuiltInCertManagementCARefresh:    "7008h",
				envBuiltInCertManagementCertValidity: "2160h",
				envBuiltInCertManagementCertRefresh:  "1728h",
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					BuiltInCertManagement: configv1alpha1.BuiltInCertManagement{
						CACertValidity: metav1.Duration{Duration: 8760 * time.Hour},
						CACertRefresh:  metav1.Duration{Duration: 7008 * time.Hour},
						CertValidity:   metav1.Duration{Duration: 2160 * time.Hour},
						CertRefresh:    metav1.Duration{Duration: 1728 * time.Hour},
					},
				},
			},
		},
		{
			name: "builtin cert management - invalid duration ignored",
			envVars: map[string]string{
				envBuiltInCertManagementCAValidity: "invalid",
			},
			initial:  configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{}, // unchanged due to invalid duration
		},
		{
			name: "builtin cert management - CA validity rejects zero duration",
			envVars: map[string]string{
				envBuiltInCertManagementCAValidity: "0s",
			},
			initial:  configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{}, // unchanged due to zero duration
		},
		{
			name: "builtin cert management - CA validity rejects negative duration",
			envVars: map[string]string{
				envBuiltInCertManagementCAValidity: "-1h",
			},
			initial:  configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{}, // unchanged due to negative duration
		},
		{
			name: "distribution",
			envVars: map[string]string{
				envDistribution: testDistributionOpenShift,
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				Distribution: testDistributionOpenShift,
			},
		},
		{
			name: "leader election - enabled",
			envVars: map[string]string{
				envLeaderElectionEnabled: "true",
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				ControllerManagerConfigurationSpec: configv1alpha1.ControllerManagerConfigurationSpec{
					LeaderElection: &componentbaseconfigv1alpha1.LeaderElectionConfiguration{
						LeaderElect: ptr.To(true),
					},
				},
			},
		},
		{
			name: "leader election - resource lock",
			envVars: map[string]string{
				envLeaderElectionResourceLock: testResourceLockLeases,
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				ControllerManagerConfigurationSpec: configv1alpha1.ControllerManagerConfigurationSpec{
					LeaderElection: &componentbaseconfigv1alpha1.LeaderElectionConfiguration{
						ResourceLock: testResourceLockLeases,
					},
				},
			},
		},
		{
			name: "leader election - resource namespace",
			envVars: map[string]string{
				envLeaderElectionResourceNamespace: testLeaderElectionNamespace,
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				ControllerManagerConfigurationSpec: configv1alpha1.ControllerManagerConfigurationSpec{
					LeaderElection: &componentbaseconfigv1alpha1.LeaderElectionConfiguration{
						ResourceNamespace: testLeaderElectionNamespace,
					},
				},
			},
		},
		{
			name: "leader election - resource name",
			envVars: map[string]string{
				envLeaderElectionResourceName: testLeaderElectionName,
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				ControllerManagerConfigurationSpec: configv1alpha1.ControllerManagerConfigurationSpec{
					LeaderElection: &componentbaseconfigv1alpha1.LeaderElectionConfiguration{
						ResourceName: testLeaderElectionName,
					},
				},
			},
		},
		{
			name: "leader election - durations",
			envVars: map[string]string{
				envLeaderElectionLeaseDuration: testLeaseDuration,
				envLeaderElectionRenewDeadline: testRenewDeadline,
				envLeaderElectionRetryPeriod:   testRetryPeriod,
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				ControllerManagerConfigurationSpec: configv1alpha1.ControllerManagerConfigurationSpec{
					LeaderElection: &componentbaseconfigv1alpha1.LeaderElectionConfiguration{
						LeaseDuration: metav1.Duration{Duration: mustParseDuration(testLeaseDuration)},
						RenewDeadline: metav1.Duration{Duration: mustParseDuration(testRenewDeadline)},
						RetryPeriod:   metav1.Duration{Duration: mustParseDuration(testRetryPeriod)},
					},
				},
			},
		},
		{
			name: "metrics bind address",
			envVars: map[string]string{
				envMetricsBindAddress: testMetricsBindAddress,
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				ControllerManagerConfigurationSpec: configv1alpha1.ControllerManagerConfigurationSpec{
					Metrics: configv1alpha1.ControllerMetrics{
						BindAddress: testMetricsBindAddress,
					},
				},
			},
		},
		{
			name: "metrics secure",
			envVars: map[string]string{
				envMetricsSecure: "true",
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				ControllerManagerConfigurationSpec: configv1alpha1.ControllerManagerConfigurationSpec{
					Metrics: configv1alpha1.ControllerMetrics{
						Secure: true,
					},
				},
			},
		},
		{
			name: "health probe bind address",
			envVars: map[string]string{
				envHealthProbeBindAddress: testHealthProbeBindAddress,
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				ControllerManagerConfigurationSpec: configv1alpha1.ControllerManagerConfigurationSpec{
					Health: configv1alpha1.ControllerHealth{
						HealthProbeBindAddress: testHealthProbeBindAddress,
					},
				},
			},
		},
		{
			name: "webhook port",
			envVars: map[string]string{
				envWebhookPort: testWebhookPort,
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				ControllerManagerConfigurationSpec: configv1alpha1.ControllerManagerConfigurationSpec{
					Webhook: configv1alpha1.ControllerWebhook{
						Port: ptr.To(mustParseInt(testWebhookPort)),
					},
				},
			},
		},
		{
			name: "webhook port - invalid value ignored",
			envVars: map[string]string{
				envWebhookPort: "invalid",
			},
			initial:  configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{}, // unchanged due to invalid port
		},
		{
			name: "metrics secure - invalid value ignored",
			envVars: map[string]string{
				envMetricsSecure: "invalid",
			},
			initial:  configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{}, // unchanged due to invalid bool
		},
		{
			name: "leader election enabled - invalid value ignored",
			envVars: map[string]string{
				envLeaderElectionEnabled: "invalid",
			},
			initial:  configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{}, // LeaderElection remains nil when no valid values are set
		},
		{
			name: "leader election - lease duration rejects zero",
			envVars: map[string]string{
				envLeaderElectionLeaseDuration: "0s",
			},
			initial:  configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{}, // LeaderElection remains nil when zero duration
		},
		{
			name: "leader election - lease duration rejects negative",
			envVars: map[string]string{
				envLeaderElectionLeaseDuration: "-15s",
			},
			initial:  configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{}, // LeaderElection remains nil when negative duration
		},
		{
			name: "combined feature gates and other env vars",
			envVars: map[string]string{
				envFeatureGates: "prometheusOperator,observability.metrics.createServiceMonitors",
				envTLSProfile:   testTLSProfileOld,
				envDistribution: testDistributionOpenShift,
			},
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				Distribution: testDistributionOpenShift,
				Gates: configv1alpha1.FeatureGates{
					TLSProfile:         testTLSProfileOld,
					PrometheusOperator: true,
					Observability: configv1alpha1.ObservabilityFeatureGates{
						Metrics: configv1alpha1.MetricsFeatureGates{
							CreateServiceMonitors: true,
						},
					},
				},
			},
		},
		// Tests for overriding existing config values
		{
			name: "override existing distribution",
			envVars: map[string]string{
				envDistribution: "openshift",
			},
			initial: configv1alpha1.ProjectConfig{
				Distribution: "kubernetes",
			},
			expected: configv1alpha1.ProjectConfig{
				Distribution: "openshift",
			},
		},
		{
			name: "override existing tls profile",
			envVars: map[string]string{
				envTLSProfile: "Modern",
			},
			initial: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					TLSProfile: "Old",
				},
			},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					TLSProfile: "Modern",
				},
			},
		},
		{
			name: "override existing metrics bind address",
			envVars: map[string]string{
				envMetricsBindAddress: ":9090",
			},
			initial: configv1alpha1.ProjectConfig{
				ControllerManagerConfigurationSpec: configv1alpha1.ControllerManagerConfigurationSpec{
					Metrics: configv1alpha1.ControllerMetrics{
						BindAddress: ":8080",
					},
				},
			},
			expected: configv1alpha1.ProjectConfig{
				ControllerManagerConfigurationSpec: configv1alpha1.ControllerManagerConfigurationSpec{
					Metrics: configv1alpha1.ControllerMetrics{
						BindAddress: ":9090",
					},
				},
			},
		},
		{
			name: "override existing leader election settings",
			envVars: map[string]string{
				envLeaderElectionEnabled:           "false",
				envLeaderElectionResourceNamespace: "new-namespace",
			},
			initial: configv1alpha1.ProjectConfig{
				ControllerManagerConfigurationSpec: configv1alpha1.ControllerManagerConfigurationSpec{
					LeaderElection: &componentbaseconfigv1alpha1.LeaderElectionConfiguration{
						LeaderElect:       ptr.To(true),
						ResourceNamespace: "old-namespace",
						ResourceName:      "old-lock",
					},
				},
			},
			expected: configv1alpha1.ProjectConfig{
				ControllerManagerConfigurationSpec: configv1alpha1.ControllerManagerConfigurationSpec{
					LeaderElection: &componentbaseconfigv1alpha1.LeaderElectionConfiguration{
						LeaderElect:       ptr.To(false),
						ResourceNamespace: "new-namespace",
						ResourceName:      "old-lock", // unchanged - not overridden
					},
				},
			},
		},
		{
			name: "override existing feature gate",
			envVars: map[string]string{
				envFeatureGates: "-httpEncryption,grpcEncryption",
			},
			initial: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					HTTPEncryption:     true,
					GRPCEncryption:     false,
					PrometheusOperator: true, // should remain unchanged
				},
			},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					HTTPEncryption:     false, // disabled by env var
					GRPCEncryption:     true,  // enabled by env var
					PrometheusOperator: true,  // unchanged
				},
			},
		},
		{
			name: "override existing cert management durations",
			envVars: map[string]string{
				envBuiltInCertManagementCAValidity: "720h",
			},
			initial: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					BuiltInCertManagement: configv1alpha1.BuiltInCertManagement{
						Enabled:        true,
						CACertValidity: metav1.Duration{Duration: 8760 * time.Hour},
						CACertRefresh:  metav1.Duration{Duration: 7008 * time.Hour},
					},
				},
			},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					BuiltInCertManagement: configv1alpha1.BuiltInCertManagement{
						Enabled:        true,                                        // unchanged
						CACertValidity: metav1.Duration{Duration: 720 * time.Hour},  // overridden
						CACertRefresh:  metav1.Duration{Duration: 7008 * time.Hour}, // unchanged
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set environment variables for the test
			for k, v := range test.envVars {
				t.Setenv(k, v)
			}

			cfg := test.initial
			ApplyEnvVars(&cfg)

			assert.Equal(t, test.expected, cfg)
		})
	}
}

func TestApplyFeatureGates(t *testing.T) {
	tests := []struct {
		name     string
		gates    string
		initial  configv1alpha1.ProjectConfig
		expected configv1alpha1.ProjectConfig
	}{
		{
			name:     "empty string",
			gates:    "",
			initial:  configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{},
		},
		{
			name:    "single gate",
			gates:   "openshift.route",
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					OpenShift: configv1alpha1.OpenShiftFeatureGates{
						OpenShiftRoute: true,
					},
				},
			},
		},
		{
			name:    "multiple gates",
			gates:   "httpEncryption,grpcEncryption",
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					HTTPEncryption: true,
					GRPCEncryption: true,
				},
			},
		},
		{
			name:  "disable gate",
			gates: "-networkPolicies",
			initial: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					NetworkPolicies: true,
				},
			},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					NetworkPolicies: false,
				},
			},
		},
		{
			name:  "mixed enable disable",
			gates: "httpEncryption,-networkPolicies",
			initial: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					NetworkPolicies: true,
				},
			},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					HTTPEncryption:  true,
					NetworkPolicies: false,
				},
			},
		},
		{
			name:    "with whitespace",
			gates:   " httpEncryption , grpcEncryption ",
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					HTTPEncryption: true,
					GRPCEncryption: true,
				},
			},
		},
		{
			name:    "empty entries ignored",
			gates:   "httpEncryption,,grpcEncryption",
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					HTTPEncryption: true,
					GRPCEncryption: true,
				},
			},
		},
		{
			name:    "unknown gate ignored",
			gates:   "unknownGate,httpEncryption",
			initial: configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					HTTPEncryption: true,
				},
			},
		},
		{
			name:     "whitespace only",
			gates:    "   ",
			initial:  configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{},
		},
		{
			name:     "only commas",
			gates:    ",,,",
			initial:  configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{},
		},
		{
			name:     "minus only ignored",
			gates:    "-",
			initial:  configv1alpha1.ProjectConfig{},
			expected: configv1alpha1.ProjectConfig{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := test.initial
			applyFeatureGates(&cfg, test.gates)
			assert.Equal(t, test.expected, cfg)
		})
	}
}

func TestParseFeatureGate(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedName    string
		expectedEnabled bool
	}{
		{
			name:            "simple gate enabled",
			input:           "httpEncryption",
			expectedName:    "httpEncryption",
			expectedEnabled: true,
		},
		{
			name:            "gate disabled with prefix",
			input:           "-httpEncryption",
			expectedName:    "httpEncryption",
			expectedEnabled: false,
		},
		{
			name:            "empty string",
			input:           "",
			expectedName:    "",
			expectedEnabled: true,
		},
		{
			name:            "lone minus returns empty name",
			input:           "-",
			expectedName:    "",
			expectedEnabled: false,
		},
		{
			name:            "whitespace only",
			input:           "   ",
			expectedName:    "",
			expectedEnabled: true,
		},
		{
			name:            "gate with leading whitespace",
			input:           "  httpEncryption",
			expectedName:    "httpEncryption",
			expectedEnabled: true,
		},
		{
			name:            "gate with trailing whitespace",
			input:           "httpEncryption  ",
			expectedName:    "httpEncryption",
			expectedEnabled: true,
		},
		{
			name:            "disabled gate with leading whitespace",
			input:           "  -httpEncryption",
			expectedName:    "httpEncryption",
			expectedEnabled: false,
		},
		{
			name:            "dotted gate name",
			input:           "openshift.route",
			expectedName:    "openshift.route",
			expectedEnabled: true,
		},
		{
			name:            "disabled dotted gate name",
			input:           "-openshift.route",
			expectedName:    "openshift.route",
			expectedEnabled: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name, enabled := parseFeatureGate(test.input)
			assert.Equal(t, test.expectedName, name)
			assert.Equal(t, test.expectedEnabled, enabled)
		})
	}
}

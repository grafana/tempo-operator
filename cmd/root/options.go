package root

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	configv1 "github.com/openshift/api/config/v1"
	openshifttls "github.com/openshift/controller-runtime-common/pkg/tls"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/internal/envconfig"
	"github.com/grafana/tempo-operator/internal/tlsprofile"
)

var errConfigFileLoading = errors.New("could not read file at path")

func loadConfigFile(scheme *runtime.Scheme, outConfig *configv1alpha1.ProjectConfig, configFile string) error {
	content, err := os.ReadFile(filepath.Clean(configFile))
	if err != nil {
		return fmt.Errorf("%w %s", errConfigFileLoading, configFile)
	}

	codecs := serializer.NewCodecFactory(scheme)

	if err = runtime.DecodeInto(codecs.UniversalDecoder(), content, outConfig); err != nil {
		return fmt.Errorf("could not decode file into runtime.Object: %w", err)
	}

	return nil
}

// LoadConfig initializes the controller configuration, optionally overriding the defaults
// from a provided configuration file.
// Returns the config, manager options, and the initial TLS profile spec that can be used
// to detect TLS profile changes and trigger graceful restart.
func LoadConfig(scheme *runtime.Scheme, configFile string) (*configv1alpha1.ProjectConfig, ctrl.Options, configv1.TLSProfileSpec, error) {
	options := ctrl.Options{Scheme: scheme}
	ctrlConfig := configv1alpha1.DefaultProjectConfig()
	var tlsProfileSpec configv1.TLSProfileSpec

	if configFile != "" {
		err := loadConfigFile(scheme, &ctrlConfig, configFile)
		if err != nil {
			return nil, options, tlsProfileSpec, fmt.Errorf("failed to parse controller manager config file: %w", err)
		}
	}

	// Apply environment variable overrides (takes precedence over config file)
	envconfig.ApplyEnvVars(&ctrlConfig)

	err := ctrlConfig.Validate()
	if err != nil {
		return nil, options, tlsProfileSpec, fmt.Errorf("controller config validation failed: %w", err)
	}

	options, tlsProfileSpec, err = mergeOptionsFromFile(options, &ctrlConfig)
	if err != nil {
		return nil, options, tlsProfileSpec, fmt.Errorf("failed to merge options from file: %w", err)
	}

	return &ctrlConfig, options, tlsProfileSpec, nil
}

func mergeOptionsFromFile(o manager.Options, cfg *configv1alpha1.ProjectConfig) (manager.Options, configv1.TLSProfileSpec, error) {
	o = setLeaderElectionConfig(o, cfg.ControllerManagerConfigurationSpec)

	if o.Metrics.BindAddress == "" && cfg.Metrics.BindAddress != "" {
		o.Metrics.BindAddress = cfg.Metrics.BindAddress
	}

	var tlsProfileSpec configv1.TLSProfileSpec
	var tlsOpts func(*tls.Config)

	if cfg.Gates.OpenShift.ClusterTLSPolicy {
		// Use the official OpenShift controller-runtime-common package for TLS profile handling
		restConfig := ctrl.GetConfigOrDie()
		k8sClient, err := client.New(restConfig, client.Options{Scheme: o.Scheme})
		if err != nil {
			return o, tlsProfileSpec, fmt.Errorf("failed to create client for TLS profile: %w", err)
		}

		// Fetch TLS profile from APIServer CR using the library function
		tlsProfileSpec, err = openshifttls.FetchAPIServerTLSProfile(context.Background(), k8sClient)
		if err != nil {
			return o, tlsProfileSpec, fmt.Errorf("failed to fetch TLS profile from cluster: %w", err)
		}

		// Convert TLS profile spec to TLS config function
		var unsupportedCiphers []string
		tlsOpts, unsupportedCiphers = openshifttls.NewTLSConfigFromProfile(tlsProfileSpec)
		if len(unsupportedCiphers) > 0 {
			ctrl.Log.WithName("setup").Info("Some ciphers from TLS profile are not supported by Go runtime",
				"unsupportedCiphers", unsupportedCiphers)
		}
	} else {
		// Use the internal tlsprofile package for non-OpenShift deployments
		tlsProfileOpts, err := tlsprofile.Get(context.Background(), cfg.Gates, nil)
		if err != nil {
			return o, tlsProfileSpec, fmt.Errorf("failed to get TLS profile: %w", err)
		}

		// Convert internal options to TLSProfileSpec for consistency
		tlsProfileSpec = configv1.TLSProfileSpec{
			Ciphers:       tlsProfileOpts.Ciphers,
			MinTLSVersion: configv1.TLSProtocolVersion(tlsProfileOpts.MinTLSVersion),
		}

		// Create TLS config function using the library with the converted spec
		tlsOpts, _ = openshifttls.NewTLSConfigFromProfile(tlsProfileSpec)
	}

	o.Metrics.SecureServing = cfg.Metrics.Secure
	if cfg.Metrics.Secure {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/metrics/filters#WithAuthenticationAndAuthorization
		o.Metrics.FilterProvider = filters.WithAuthenticationAndAuthorization
		o.Metrics.TLSOpts = []func(*tls.Config){tlsOpts}
	}
	if o.HealthProbeBindAddress == "" && cfg.Health.HealthProbeBindAddress != "" {
		o.HealthProbeBindAddress = cfg.Health.HealthProbeBindAddress
	}

	if cfg.Webhook.Port != nil {
		o.WebhookServer = webhook.NewServer(webhook.Options{
			Port:    *cfg.Webhook.Port,
			TLSOpts: []func(*tls.Config){tlsOpts},
		})
	}

	return o, tlsProfileSpec, nil
}

func setLeaderElectionConfig(o manager.Options, obj configv1alpha1.ControllerManagerConfigurationSpec) manager.Options {
	if obj.LeaderElection == nil {
		// The source does not have any configuration; noop
		return o
	}

	if !o.LeaderElection && obj.LeaderElection.LeaderElect != nil {
		o.LeaderElection = *obj.LeaderElection.LeaderElect
	}

	if o.LeaderElectionResourceLock == "" && obj.LeaderElection.ResourceLock != "" {
		o.LeaderElectionResourceLock = obj.LeaderElection.ResourceLock
	}

	if o.LeaderElectionNamespace == "" && obj.LeaderElection.ResourceNamespace != "" {
		o.LeaderElectionNamespace = obj.LeaderElection.ResourceNamespace
	}

	if o.LeaderElectionID == "" && obj.LeaderElection.ResourceName != "" {
		o.LeaderElectionID = obj.LeaderElection.ResourceName
	}

	if o.LeaseDuration == nil && !reflect.DeepEqual(obj.LeaderElection.LeaseDuration, metav1.Duration{}) {
		o.LeaseDuration = &obj.LeaderElection.LeaseDuration.Duration
	}

	if o.RenewDeadline == nil && !reflect.DeepEqual(obj.LeaderElection.RenewDeadline, metav1.Duration{}) {
		o.RenewDeadline = &obj.LeaderElection.RenewDeadline.Duration
	}

	if o.RetryPeriod == nil && !reflect.DeepEqual(obj.LeaderElection.RetryPeriod, metav1.Duration{}) {
		o.RetryPeriod = &obj.LeaderElection.RetryPeriod.Duration
	}

	return o
}

package root

import (
	"errors"
	"fmt"

	"os"
	"path/filepath"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
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
func LoadConfig(scheme *runtime.Scheme, configFile string) (*configv1alpha1.ProjectConfig, ctrl.Options, error) {
	options := ctrl.Options{Scheme: scheme}
	ctrlConfig := configv1alpha1.DefaultProjectConfig()

	if configFile == "" {
		return &ctrlConfig, options, nil
	}

	err := loadConfigFile(scheme, &ctrlConfig, configFile)
	if err != nil {
		return nil, options, fmt.Errorf("failed to parse controller manager config file: %w", err)
	}

	err = ctrlConfig.Validate()
	if err != nil {
		return nil, options, fmt.Errorf("controller config validation failed: %w", err)
	}

	options = mergeOptionsFromFile(options, &ctrlConfig)

	return &ctrlConfig, options, nil
}

func mergeOptionsFromFile(o manager.Options, cfg *configv1alpha1.ProjectConfig) manager.Options {
	o = setLeaderElectionConfig(o, cfg.ControllerManagerConfigurationSpec)

	if o.Metrics.BindAddress == "" && cfg.Metrics.BindAddress != "" {
		o.Metrics.BindAddress = cfg.Metrics.BindAddress
	}

	o.Metrics.SecureServing = cfg.Metrics.Secure
	if cfg.Metrics.Secure {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/metrics/filters#WithAuthenticationAndAuthorization
		o.Metrics.FilterProvider = filters.WithAuthenticationAndAuthorization
	}
	if o.HealthProbeBindAddress == "" && cfg.Health.HealthProbeBindAddress != "" {
		o.HealthProbeBindAddress = cfg.Health.HealthProbeBindAddress
	}

	if cfg.Webhook.Port != nil {
		o.WebhookServer = webhook.NewServer(webhook.Options{
			Port: *cfg.Webhook.Port,
		})
	}

	return o
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

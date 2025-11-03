package start

import (
	"context"
	"fmt"
	"os"
	"runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/cmd/root"
	controllers "github.com/grafana/tempo-operator/internal/controller/tempo"
	"github.com/grafana/tempo-operator/internal/crdmetrics"
	"github.com/grafana/tempo-operator/internal/version"
	"github.com/grafana/tempo-operator/internal/webhooks"
	//+kubebuilder:scaffold:imports
)

func start(c *cobra.Command, args []string) {
	rootCmdConfig := c.Context().Value(root.RootConfigKey{}).(root.RootConfig)
	ctrlConfig, options := rootCmdConfig.CtrlConfig, rootCmdConfig.Options
	setupLog := ctrl.Log.WithName("setup")
	version := version.Get()

	options.PprofBindAddress, _ = c.Flags().GetString("pprof-addr")

	certDir, _ := c.Flags().GetString("metrics-tls-cert-dir")
	if certDir != "" {
		options.Metrics.CertDir = certDir
		options.Metrics.CertName, _ = c.Flags().GetString("metrics-tls-cert-file")
		options.Metrics.KeyName, _ = c.Flags().GetString("metrics-tls-private-key-file")
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	err = addDependencies(mgr, ctrlConfig)
	if err != nil {
		setupLog.Error(err, "failed to upgrade TempoStack instances")
		os.Exit(1)
	}

	if ctrlConfig.Gates.BuiltInCertManagement.Enabled {
		if err = (&controllers.CertRotationReconciler{
			Client:       mgr.GetClient(),
			Scheme:       mgr.GetScheme(),
			FeatureGates: ctrlConfig.Gates,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "certrotation")
			os.Exit(1)
		}

		if err = (&controllers.CertRotationMonolithicReconciler{
			Client:       mgr.GetClient(),
			Scheme:       mgr.GetScheme(),
			FeatureGates: ctrlConfig.Gates,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "certrotationmonolithic")
			os.Exit(1)
		}
	}

	if err = (&controllers.TempoStackReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Recorder:   mgr.GetEventRecorderFor("tempostack-controller"),
		CtrlConfig: ctrlConfig,
		Version:    version,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TempoStack")
		os.Exit(1)
	}

	if err = (&controllers.TempoMonolithicReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Recorder:   mgr.GetEventRecorderFor("tempomonolithic-controller"),
		CtrlConfig: ctrlConfig,
		Version:    version,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TempoMonolithic")
		os.Exit(1)
	}

	enableWebhooks := os.Getenv("ENABLE_WEBHOOKS") != "false"
	if enableWebhooks {
		if err = (&webhooks.TempoStackWebhook{}).SetupWebhookWithManager(mgr, ctrlConfig); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "TempoStack")
			os.Exit(1)
		}
		if err = (&webhooks.TempoMonolithicWebhook{}).SetupWebhookWithManager(mgr, ctrlConfig); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "TempoMonolithic")
			os.Exit(1)
		}
	}
	//+kubebuilder:scaffold:builder

	healthCheck := healthz.Ping
	if enableWebhooks {
		healthCheck = mgr.GetWebhookServer().StartedChecker()
	}
	if err := mgr.AddHealthzCheck("healthz", healthCheck); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthCheck); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("Starting Tempo Operator",
		"build-date", version.BuildDate,
		"revision", version.Revision,
		"tempo-operator", version.OperatorVersion,
		"tempo", version.TempoVersion,
		"tempo-query", version.TempoQueryVersion,
		"default-tempo-image", rootCmdConfig.CtrlConfig.DefaultImages.Tempo,
		"default-tempo-query-image", rootCmdConfig.CtrlConfig.DefaultImages.TempoQuery,
		"default-tempo-gateway-image", rootCmdConfig.CtrlConfig.DefaultImages.TempoGateway,
		"default-tempo-gateway-opa-image", rootCmdConfig.CtrlConfig.DefaultImages.TempoGatewayOpa,
		"default-network-policies", ctrlConfig.Gates.NetworkPolicies,
		"go-version", version.GoVersion,
		"go-arch", runtime.GOARCH,
		"go-os", runtime.GOOS,
	)

	if err := crdmetrics.Bootstrap(mgr.GetClient()); err != nil {
		setupLog.Error(err, "problem init crd metrics")
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func addDependencies(mgr ctrl.Manager, ctrlConfig configv1alpha1.ProjectConfig) error {
	err := mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		reconciler := &controllers.OperatorReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
			Config: mgr.GetConfig(),
		}

		// log error but do not fail operator startup if operator reconcile fails
		// operator reconcile is only used for creating ServiceMonitor and PrometheusRules of the operator itself
		err := reconciler.Reconcile(ctx, ctrlConfig)
		if err != nil {
			ctrl.LoggerFrom(ctx).WithName("operator-reconcile").Error(err, "cannot reconcile operator")
		}
		return nil
	}))
	if err != nil {
		return fmt.Errorf("failed to setup operator reconciler: %w", err)
	}

	return nil
}

// NewStartCommand returns a new start command.
func NewStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the Tempo operator",
		Run:   start,
	}
	cmd.Flags().String("pprof-addr", "", "The address the pprof server binds to. Default is empty string which disables the pprof server.")
	cmd.Flags().String("metrics-tls-cert-dir", "", "TLS certificate used by metrics server")
	cmd.Flags().String("metrics-tls-cert-file", "tls.crt", "TLS certificate used by metrics server")
	cmd.Flags().String("metrics-tls-private-key-file", "tls.key", "TLS key used by metrics server")
	return cmd
}

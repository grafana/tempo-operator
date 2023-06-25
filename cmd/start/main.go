package start

import (
	"context"
	"os"
	"runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	tempov1alpha1 "github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/cmd"
	controllers "github.com/grafana/tempo-operator/controllers/tempo"
	"github.com/grafana/tempo-operator/internal/upgrade"
	"github.com/grafana/tempo-operator/internal/version"
	//+kubebuilder:scaffold:imports
)

func start(c *cobra.Command, args []string) {
	rootCmdConfig := c.Context().Value(cmd.RootConfigKey{}).(cmd.RootConfig)
	ctrlConfig, options := rootCmdConfig.CtrlConfig, rootCmdConfig.Options
	setupLog := ctrl.Log.WithName("setup")
	version := version.Get()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// run the upgrade mechanism once the manager is ready
	err = mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		upgrade := &upgrade.Upgrade{
			Client:     mgr.GetClient(),
			Recorder:   mgr.GetEventRecorderFor("tempo-upgrade"),
			CtrlConfig: ctrlConfig,
			Version:    version,
			Log:        ctrl.Log.WithName("upgrade"),
		}
		return upgrade.TempoStacks(ctx)
	}))
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
	}

	if err = (&controllers.TempoStackReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		FeatureGates: ctrlConfig.Gates,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TempoStack")
		os.Exit(1)

	}

	enableWebhooks := os.Getenv("ENABLE_WEBHOOKS") != "false"
	if enableWebhooks {
		if err = (&tempov1alpha1.TempoStack{}).SetupWebhookWithManager(mgr, ctrlConfig); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "TempoStack")
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
		"go-version", version.GoVersion,
		"go-arch", runtime.GOARCH,
		"go-os", runtime.GOOS,
	)

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// NewStartCommand returns a new start command.
func NewStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the Tempo operator",
		Run:   start,
	}
}

package root

import (
	"context"

	grafanav1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	configv1 "github.com/openshift/api/config/v1"
	openshiftoperatorv1 "github.com/openshift/api/operator/v1"
	routev1 "github.com/openshift/api/route/v1"
	cloudcredentialv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

var (
	scheme = runtime.NewScheme()
)

// RootConfigKey contains the key to RootConfig in the context object.
type RootConfigKey struct{}

// RootConfig contains configuration relevant for all commands.
type RootConfig struct {
	Options    ctrl.Options
	CtrlConfig configv1alpha1.ProjectConfig
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(tempov1alpha1.AddToScheme(scheme))
	utilruntime.Must(configv1alpha1.AddToScheme(scheme))
	utilruntime.Must(routev1.Install(scheme))
	utilruntime.Must(cloudcredentialv1.Install(scheme))
	utilruntime.Must(openshiftoperatorv1.Install(scheme))
	utilruntime.Must(configv1.Install(scheme))
	utilruntime.Must(monitoringv1.AddToScheme(scheme))
	utilruntime.Must(grafanav1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func readConfig(cmd *cobra.Command, configFile string) error {
	// default controller configuration
	ctrlCfg, options, err := LoadConfig(scheme, configFile)
	if err != nil {
		return err
	}

	cmd.SetContext(context.WithValue(cmd.Context(), RootConfigKey{}, RootConfig{options, *ctrlCfg}))
	return nil
}

// NewRootCommand creates a new cobra root command.
func NewRootCommand() *cobra.Command {
	var configFile string

	rootCmd := &cobra.Command{
		Use:          "tempo-operator",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return readConfig(cmd, configFile)
		},
	}
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")

	return rootCmd
}

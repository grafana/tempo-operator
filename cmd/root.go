package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	configtempov1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
	tempov1alpha1 "github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

var (
	scheme = runtime.NewScheme()
)

// RootConfigKey contains the key to RootConfig in the context object.
type RootConfigKey struct{}

// RootConfig contains configuration relevant for all commands.
type RootConfig struct {
	Options    ctrl.Options
	CtrlConfig configtempov1alpha1.ProjectConfig
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(tempov1alpha1.AddToScheme(scheme))
	utilruntime.Must(configtempov1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func readConfig(cmd *cobra.Command, configFile string) error {
	var err error

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrlConfig := configtempov1alpha1.ProjectConfig{}
	options := ctrl.Options{Scheme: scheme}
	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile).OfKind(&ctrlConfig))
		if err != nil {
			return fmt.Errorf("unable to load the config file: %w", err)
		}
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	cmd.SetContext(context.WithValue(cmd.Context(), RootConfigKey{}, RootConfig{options, ctrlConfig}))
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

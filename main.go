package main

import (
	"flag"
	"os"

	"go.uber.org/zap/zapcore"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/os-observability/tempo-operator/cmd"
	"github.com/os-observability/tempo-operator/cmd/generate"
	"github.com/os-observability/tempo-operator/cmd/start"
	"github.com/os-observability/tempo-operator/cmd/version"
)

func main() {
	rootCmd := cmd.NewRootCommand()
	rootCmd.AddCommand(start.NewStartCommand())
	rootCmd.AddCommand(generate.NewGenerateCommand())
	rootCmd.AddCommand(version.NewVersionCommand())

	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.ISO8601TimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	// pass remaining flags (excluding zap flags) to spf13/cobra commands
	args := flag.Args()
	rootCmd.SetArgs(args)

	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

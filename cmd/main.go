package main

import (
	"flag"
	"os"

	"github.com/grafana/tempo-operator/cmd/generate"
	"github.com/grafana/tempo-operator/cmd/root"
	"github.com/grafana/tempo-operator/cmd/start"
	"github.com/grafana/tempo-operator/cmd/version"
	"github.com/grafana/tempo-operator/internal/logging"
)

func main() {
	rootCmd := root.NewRootCommand()
	rootCmd.AddCommand(start.NewStartCommand())
	rootCmd.AddCommand(generate.NewGenerateCommand())
	rootCmd.AddCommand(version.NewVersionCommand())

	logging.SetupLogging()

	// pass remaining flags (excluding zap flags) to spf13/cobra commands
	args := flag.Args()
	rootCmd.SetArgs(args)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

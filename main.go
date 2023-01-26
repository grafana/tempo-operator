package main

import (
	"os"

	"github.com/os-observability/tempo-operator/cmd"
	"github.com/os-observability/tempo-operator/cmd/generate"
	"github.com/os-observability/tempo-operator/cmd/start"
)

func main() {
	rootCmd := cmd.NewRootCommand()
	rootCmd.AddCommand(start.NewStartCommand())
	rootCmd.AddCommand(generate.NewGenerateCommand())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

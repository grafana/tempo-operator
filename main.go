package main

import (
	"flag"
	"os"

	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapio"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/go-logr/zapr"
	"github.com/os-observability/tempo-operator/cmd"
	"github.com/os-observability/tempo-operator/cmd/generate"
	"github.com/os-observability/tempo-operator/cmd/start"
	"github.com/os-observability/tempo-operator/cmd/version"
)

// redirect log messages produced by k8s.io/klog (e.g. used by client-go) to zap.Logger
func redirectKlog(logger *uzap.Logger) {
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	// logtostderr bypasses SetOutputBySeverity()
	klogFlags.Set("logtostderr", "false")

	// never print to stderr directly
	klogFlags.Set("stderrthreshold", "999")

	// skip headers (level, time, etc.), because logr captures them already
	klogFlags.Set("skip_headers", "true")

	// only write to the specified log level, not to every lower severity level
	klogFlags.Set("one_output", "true")

	klog.SetOutputBySeverity("INFO", &zapio.Writer{Log: logger, Level: zapcore.InfoLevel})
	klog.SetOutputBySeverity("WARNING", &zapio.Writer{Log: logger, Level: zapcore.WarnLevel})
	klog.SetOutputBySeverity("ERROR", &zapio.Writer{Log: logger, Level: zapcore.ErrorLevel})
	klog.SetOutputBySeverity("FATAL", &zapio.Writer{Log: logger, Level: zapcore.FatalLevel})
}

func setupLogging() {
	opts := zap.Options{
		TimeEncoder: zapcore.RFC3339NanoTimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	zapLogger := zap.NewRaw(zap.UseFlagOptions(&opts))
	logger := zapr.NewLogger(zapLogger)
	ctrl.SetLogger(logger)
	redirectKlog(zapLogger)
}

func main() {
	rootCmd := cmd.NewRootCommand()
	rootCmd.AddCommand(start.NewStartCommand())
	rootCmd.AddCommand(generate.NewGenerateCommand())
	rootCmd.AddCommand(version.NewVersionCommand())

	setupLogging()

	// pass remaining flags (excluding zap flags) to spf13/cobra commands
	args := flag.Args()
	rootCmd.SetArgs(args)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

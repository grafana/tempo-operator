package logging

import (
	"flag"

	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapio"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/go-logr/zapr"
)

// redirectKlog redirects log messages produced by k8s.io/klog (e.g. used by client-go) to zap.Logger.
func redirectKlog(logger *uzap.Logger) {
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	// logtostderr bypasses SetOutputBySeverity()
	_ = klogFlags.Set("logtostderr", "false")

	// never print to stderr directly
	_ = klogFlags.Set("stderrthreshold", "999")

	// skip headers (level, time, etc.), because logr captures them already
	_ = klogFlags.Set("skip_headers", "true")

	// only write to the specified log level, not to every lower severity level
	_ = klogFlags.Set("one_output", "true")

	klog.SetOutputBySeverity("INFO", &zapio.Writer{Log: logger, Level: zapcore.InfoLevel})
	klog.SetOutputBySeverity("WARNING", &zapio.Writer{Log: logger, Level: zapcore.WarnLevel})
	klog.SetOutputBySeverity("ERROR", &zapio.Writer{Log: logger, Level: zapcore.ErrorLevel})
	klog.SetOutputBySeverity("FATAL", &zapio.Writer{Log: logger, Level: zapcore.FatalLevel})
}

// SetupLogging configures zap from CLI flags.
func SetupLogging() {
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

package util

import (
	"flag"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/kelseyhightower/envconfig"
	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Logger struct {
	cfg LogConfig
}

type LogConfig struct {
	LogMode      string `envconfig:"LOG_MODE" default:"production"`
	LogEncoding  string `envconfig:"LOG_ENCODING"`
	LogLevel     string `envconfig:"LOG_LEVEL"`
	LogVerbosity int8   `envconfig:"LOG_VERBOSITY" default:"0"`
}

// New will return a Logger configured with the LOG_* environment variables
// and the supported --zap* flags passed to the operator command line
func (l Logger) New() logr.Logger {

	if err := envconfig.Process("log", &l.cfg); err != nil {
		fmt.Fprintf(os.Stderr, "unable to get log env variables")
	}

	opts := zap.Options{}
	encoderConfig := zapcore.EncoderConfig{}

	// Development configures the logger to use a Zap development config
	// (stacktraces on warnings, no sampling), otherwise a Zap production
	// config will be used (stacktraces on errors, sampling).
	if l.cfg.LogMode == "production" {
		opts.Development = false
		encoderConfig = uzap.NewProductionEncoderConfig()
	} else {
		opts.Development = true
		encoderConfig = uzap.NewDevelopmentEncoderConfig()
	}

	// Encoder configures how Zap will encode the output.  Defaults to
	// console when Development is true and JSON otherwise
	switch string(l.cfg.LogEncoding) {
	case "json", "JSON":
		opts.Encoder = zapcore.NewJSONEncoder(encoderConfig)
	case "console", "CONSOLE":
		opts.Encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Log level configures the zap log level, from `info` to `fatal`
	if l.cfg.LogLevel != "" {
		lvl := zapcore.Level(0)
		if err := lvl.UnmarshalText([]byte(l.cfg.LogLevel)); err != nil {
			fmt.Fprint(os.Stderr, err.Error())
		}
		opts.Level = lvl
	}

	// Level enforces debug verbosity for logs when level is LOG_VERBOSITY is set
	if l.cfg.LogVerbosity > 0 {
		opts.Level = zapcore.Level(0 - l.cfg.LogVerbosity)
	}

	// Allow also commandline based log configuration
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	return zap.New(zap.UseFlagOptions(&opts))
}

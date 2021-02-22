// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package ctxlog

import (
	"context"
	"flag"
	"io"

	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	defaultLogLevel = zapcore.DebugLevel

	flagOptions = &zap.Options{
		Level: &defaultLogLevel,
	}
)

// CustomZapFlagSet creates a flag.FlagSet containing the zap logger default flags to configure the zap options and it
// includes a custom flag `--zap-level` for backwards compatibility reasons
func CustomZapFlagSet() *flag.FlagSet {
	f := flag.NewFlagSet("zap", flag.ExitOnError)
	flagOptions.BindFlags(f)

	// Add --zap-level for backwards compatibility and hard-wire it against the log level settings. Remove this line and
	// the defaultLogLevel as soon as enough time has passed to fix the flag usage.
	f.Var(&defaultLogLevel, "zap-level", "Deprecated: Please use --zap-log-level instead; set log level")

	return f
}

// NewLogger returns a new Logger instance
// by using the controller-runtime log/zap logging
// implementation
func NewLogger(name string) logr.Logger {
	l := zap.New(zap.UseFlagOptions(flagOptions))
	logf.SetLogger(l)

	return l.WithName(name)
}

// NewLoggerTo returns a new Logger which logs
// to a given destination.
func NewLoggerTo(destWriter io.Writer, name string) logr.Logger {
	l := zap.New(zap.UseFlagOptions(flagOptions), zap.WriteTo(destWriter))
	logf.SetLogger(l)

	return l.WithName(name)
}

// Error returns an ERROR level log from an specified context
func Error(ctx context.Context, err error, msg string, v ...interface{}) {
	l := ExtractLogger(ctx)
	l.Error(err, msg, v...)
}

// Debug returns an DEBUG level log from an specified context
func Debug(ctx context.Context, msg string, v ...interface{}) {
	l := ExtractLogger(ctx)
	l.V(1).Info(msg, v...)
}

// Info returns an INFO level log from an specified context
func Info(ctx context.Context, msg string, v ...interface{}) {
	l := ExtractLogger(ctx)
	l.Info(msg, v...)
}

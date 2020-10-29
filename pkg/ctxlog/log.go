// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package ctxlog

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// NewLogger returns a new Logger instance
// by using the operator-sdk log/zap logging
// implementation
func NewLogger(name string) logr.Logger {
	l := zap.Logger()

	logf.SetLogger(l)

	l = l.WithName(name)

	return l
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

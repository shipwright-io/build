// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package ctxlog

import (
	"context"

	"github.com/go-logr/logr"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type contextLogger struct{}

var (
	loggerKey = &contextLogger{}
)

// NewParentContext returns a new context from the
// parent context.Background one. This new context
// stores our logger implementation
func NewParentContext(log logr.Logger) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, loggerKey, log)
	return ctx
}

// NewContext returns a new child context based on our logger
// key(loggerKey). This function is useful for spawning children
// context with a particular logging name for each controller
func NewContext(ctx context.Context, name string) context.Context {
	l := ExtractLogger(ctx)

	l = l.WithName(name)

	return context.WithValue(ctx, loggerKey, l)
}

// ExtractLogger returns a logger based on the loggerKey
// This function retrieves from an existing context the value,
// which in this case is an instance of our logger
func ExtractLogger(ctx context.Context) logr.Logger {
	log, ok := ctx.Value(loggerKey).(logr.Logger)
	if !ok || log.GetSink() == nil {
		if logger, err := logr.FromContext(ctx); err == nil {
			log = logger
		}
		if log.GetSink() == nil {
			log = log.WithSink(logf.NullLogSink{})
		}
	}
	return log
}

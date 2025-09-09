// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"bytes"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	buildconfig "github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/controller"
	"github.com/shipwright-io/build/pkg/ctxlog"
)

// StartBuildControllers initialize an operator as if being call from main,
// but it disables the prometheus metrics and leader election. This intended
// to for testing.
func StartBuildControllers() (*bytes.Buffer, error) {
	// Check for BUILDRUN_EXECUTOR environment variable
	executor := os.Getenv("BUILDRUN_EXECUTOR")
	return StartBuildControllersWithExecutor(executor)
}

// StartBuildControllersWithExecutor initialize an operator with a specific executor type.
// If executor is empty, it will use the default or environment variable value.
func StartBuildControllersWithExecutor(executor string) (*bytes.Buffer, error) {
	logBuffer := &bytes.Buffer{}
	l := ctxlog.NewLoggerTo(logBuffer, "controller")
	ctx := ctxlog.NewParentContext(l)

	c := buildconfig.NewDefaultConfig()

	// Set executor if provided
	if executor != "" {
		c.BuildrunExecutor = executor
	}

	// read configuration from environment variables, especially the GIT_CONTAINER_IMAGE
	c.SetConfigFromEnv()

	_, restConfig, err := KubeConfig()
	if err != nil {
		return nil, err
	}

	mgr, err := controller.NewManager(ctx, c, restConfig, manager.Options{
		LeaderElection: false,
	})
	if err != nil {
		return nil, err
	}

	go func() {
		// set stopChan with the channel for future closing
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	return logBuffer, nil
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"

	buildconfig "github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/controller"
)

// StartBuildControllers initialize an operator as if being call from main,
// but it disables the prometheus metrics and leader election. This intended
// to for testing.
func (t *TestBuild) StartBuildControllers() error {
	c := buildconfig.NewDefaultConfig()

	// read configuration from environment variables, especially the GIT_CONTAINER_IMAGE
	c.SetConfigFromEnv()

	mgr, err := controller.NewManager(t.Context, c, t.KubeConfig, manager.Options{
		Namespace:          t.Namespace,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	if err != nil {
		return err
	}

	go func() {
		// set stopChan with the channel for future closing
		err := mgr.Start(t.Context)
		if err != nil {
			panic(err)
		}
	}()

	return nil
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	buildconfig "github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// StartBuildOperator initialize an operator as if being call from main,
// but it disables the prometheus metrics and leader election. This intended
// to for testing.
func (t *TestBuild) StartBuildOperator() (chan struct{}, error) {
	c := buildconfig.NewDefaultConfig()

	mgr, err := controller.NewManager(context.Background(), c, t.KubeConfig, manager.Options{
		Namespace:          t.Namespace,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})

	if err != nil {
		return nil, err
	}

	stopChan := make(chan struct{})
	go func() {
		// set stopChan with the channel for future closing
		err := mgr.Start(stopChan)
		if err != nil {
			panic(err)
		}
	}()

	return stopChan, err
}

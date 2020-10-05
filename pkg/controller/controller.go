// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"

	"github.com/shipwright-io/build/pkg/config"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(context.Context, *config.Config, manager.Manager) error

// AddToManager adds all Controllers to the Manager
func AddToManager(ctx context.Context, c *config.Config, m manager.Manager) error {
	for _, f := range AddToManagerFuncs {
		if err := f(ctx, c, m); err != nil {
			return err
		}
	}
	return nil
}

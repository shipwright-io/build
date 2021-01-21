// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/shipwright-io/build/pkg/apis"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
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

// NewManager add all the controllers to the manager and register the required schemes
func NewManager(ctx context.Context, config *config.Config, cfg *rest.Config, options manager.Options) (manager.Manager, error) {
	mgr, err := manager.New(cfg, options)
	if err != nil {
		return nil, err
	}

	ctxlog.Info(ctx, "Registering Components.")

	if err := pipelinev1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}

	// Setup all Controllers
	if err := AddToManager(ctx, config, mgr); err != nil {
		return nil, err
	}

	return mgr, nil
}

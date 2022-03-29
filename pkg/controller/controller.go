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
	"github.com/shipwright-io/build/pkg/reconciler/build"
	"github.com/shipwright-io/build/pkg/reconciler/build_limit_cleanup"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun_ttl_cleanup"
	"github.com/shipwright-io/build/pkg/reconciler/buildstrategy"
	"github.com/shipwright-io/build/pkg/reconciler/clusterbuildstrategy"
)

// NewManager add all the controllers to the manager and register the required schemes
func NewManager(ctx context.Context, config *config.Config, cfg *rest.Config, options manager.Options) (manager.Manager, error) {
	mgr, err := manager.New(cfg, options)
	if err != nil {
		return nil, err
	}

	if config.KubeAPIOptions.Burst > 0 {
		mgr.GetConfig().Burst = config.KubeAPIOptions.Burst
	}
	if config.KubeAPIOptions.QPS > 0 {
		mgr.GetConfig().QPS = float32(config.KubeAPIOptions.QPS)
	}

	ctxlog.Info(ctx, "Registering Components.")

	if err := pipelinev1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}

	// Add Reconcilers.
	if err := build.Add(ctx, config, mgr); err != nil {
		return nil, err
	}

	if err := buildrun.Add(ctx, config, mgr); err != nil {
		return nil, err
	}

	if err := buildstrategy.Add(ctx, config, mgr); err != nil {
		return nil, err
	}

	if err := clusterbuildstrategy.Add(ctx, config, mgr); err != nil {
		return nil, err
	}

	if err := build_limit_cleanup.Add(ctx, config, mgr); err != nil {
		return nil, err
	}

	if err := buildrun_ttl_cleanup.Add(ctx, config, mgr); err != nil {
		return nil, err
	}

	return mgr, nil
}

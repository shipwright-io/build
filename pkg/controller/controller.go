// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"

	"github.com/shipwright-io/build/pkg/apis"
	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/shipwright-io/build/pkg/reconciler/build"
	"github.com/shipwright-io/build/pkg/reconciler/buildlimitcleanup"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun"
	"github.com/shipwright-io/build/pkg/reconciler/buildrunttlcleanup"
	"github.com/shipwright-io/build/pkg/reconciler/buildstrategy"
	"github.com/shipwright-io/build/pkg/reconciler/clusterbuildstrategy"
)

// NewManager add all the controllers to the manager and register the required schemes
func NewManager(ctx context.Context, config *config.Config, cfg *rest.Config, options manager.Options) (manager.Manager, error) {
	// Setup a scheme
	options.Scheme = k8sruntime.NewScheme()
	if err := corev1.AddToScheme(options.Scheme); err != nil {
		return nil, err
	}
	if err := pipelineapi.AddToScheme(options.Scheme); err != nil {
		return nil, err
	}
	if err := apis.AddToScheme(options.Scheme); err != nil {
		return nil, err
	}

	// Configure the cache
	buildRunLabelExistsSelector, err := labels.Parse(buildv1beta1.LabelBuildRun)
	if err != nil {
		return nil, err
	}

	options.Cache = cache.Options{
		ByObject: map[client.Object]cache.ByObject{
			&corev1.Pod{}: {
				Label: buildRunLabelExistsSelector,
			},
			&pipelineapi.TaskRun{}: {
				Label: buildRunLabelExistsSelector,
			},
		},
	}

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

	if err := buildlimitcleanup.Add(ctx, config, mgr); err != nil {
		return nil, err
	}

	if err := buildrunttlcleanup.Add(ctx, config, mgr); err != nil {
		return nil, err
	}

	return mgr, nil
}

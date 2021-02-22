// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package clusterbuildstrategy

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
)

// blank assignment to verify that ReconcileClusterBuildStrategy implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileClusterBuildStrategy{}

// ReconcileClusterBuildStrategy reconciles a ClusterBuildStrategy object
type ReconcileClusterBuildStrategy struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	ctx    context.Context
	config *config.Config
	client client.Client
	scheme *runtime.Scheme
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(ctx context.Context, c *config.Config, mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileClusterBuildStrategy{
		ctx:    ctx,
		config: c,
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// Reconcile reads that state of the cluster for a ClusterBuildStrategy object and makes changes based on the state read
// and what is in the BuildStrategy.Spec
func (r *ReconcileClusterBuildStrategy) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(r.ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Info(ctx, "reconciling ClusterBuildStrategy", "name", request.Name)
	return reconcile.Result{}, nil
}

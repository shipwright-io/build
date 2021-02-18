// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package clusterbuildstrategy

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/shipwright-io/build/pkg/ctxlog"
)

// Reconcile reads that state of the cluster for a ClusterBuildStrategy object and makes changes based on the state read
// and what is in the BuildStrategy.Spec
func (r *ReconcileClusterBuildStrategy) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(r.ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Info(ctx, "reconciling ClusterBuildStrategy", "name", request.Name)
	return reconcile.Result{}, nil
}

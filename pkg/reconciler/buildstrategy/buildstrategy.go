// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildstrategy

import (
	"context"

	"github.com/shipwright-io/build/pkg/ctxlog"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconcile reads that state of the cluster for a BuildStrategy object and makes changes based on the state read
// and what is in the BuildStrategy.Spec
func (r *ReconcileBuildStrategy) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(r.ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Info(ctx, "reconciling BuildStrategy", "namespace", request.Namespace, "name", request.Name)
	return reconcile.Result{}, nil
}

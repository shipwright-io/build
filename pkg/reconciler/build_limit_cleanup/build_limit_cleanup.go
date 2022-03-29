// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package build_limit_cleanup

import (
	"context"
	"sort"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileBuild reconciles a Build object
type ReconcileBuild struct {
	/* This client, initialized using mgr.Client() above, is a split client
	   that reads objects from the cache and writes to the apiserver */
	config                *config.Config
	client                client.Client
	scheme                *runtime.Scheme
	setOwnerReferenceFunc setOwnerReferenceFunc
}

func NewReconciler(c *config.Config, mgr manager.Manager, ownerRef setOwnerReferenceFunc) reconcile.Reconciler {
	return &ReconcileBuild{
		config:                c,
		client:                mgr.GetClient(),
		scheme:                mgr.GetScheme(),
		setOwnerReferenceFunc: ownerRef,
	}
}

/* Reconciler finds retentions fields in builds and makes sure the
   number of corresponding buildruns adhere to these limits */
func (r *ReconcileBuild) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Debug(ctx, "Start reconciling build-limit-cleanup", namespace, request.Namespace, name, request.Name)

	b := &build.Build{}
	err := r.client.Get(ctx, request.NamespacedName, b)

	if err != nil {
		if apierrors.IsNotFound(err) {
			ctxlog.Debug(ctx, "Finish reconciling build-limit-cleanup. Build was not found", namespace, request.Namespace, name, request.Name)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	lbls := map[string]string{
		build.LabelBuild: b.Name,
	}
	opts := client.ListOptions{
		Namespace:     b.Namespace,
		LabelSelector: labels.SelectorFromSet(lbls),
	}
	allBuildRuns := &build.BuildRunList{}
	r.client.List(ctx, allBuildRuns, &opts)
	if len(allBuildRuns.Items) == 0 {
		return reconcile.Result{}, nil
	}

	var buildRunFailed []build.BuildRun
	var buildRunSucceeded []build.BuildRun

	// Sort buildruns into successful ones and failed ones
	for _, br := range allBuildRuns.Items {
		condition := br.Status.GetCondition(build.Succeeded)
		if condition != nil {
			if condition.Status == corev1.ConditionFalse {
				buildRunFailed = append(buildRunFailed, br)
			} else if condition.Status == corev1.ConditionTrue {
				buildRunSucceeded = append(buildRunSucceeded, br)
			}
		}
	}

	// Check limits and delete oldest buildruns if limit is reached.
	if b.Spec.Retention.SucceededLimit != nil {
		if len(buildRunSucceeded) > int(*b.Spec.Retention.SucceededLimit) {
			sort.Slice(buildRunSucceeded, func(i, j int) bool {
				return buildRunSucceeded[i].Status.CompletionTime.Before(buildRunSucceeded[j].Status.CompletionTime)
			})
			lenOfList := len(buildRunSucceeded)
			for i := 0; lenOfList-i > int(*b.Spec.Retention.SucceededLimit); i += 1 {
				ctxlog.Info(ctx, "Deleting succeeded buildrun as cleanup limit has been reached.", namespace, request.Namespace, name, buildRunSucceeded[i].Name)
				deleteBuildRunErr := r.client.Delete(ctx, &buildRunSucceeded[i], &client.DeleteOptions{})
				if deleteBuildRunErr != nil {
					if apierrors.IsNotFound(deleteBuildRunErr) {
						return reconcile.Result{}, nil
					}
					ctxlog.Debug(ctx, "Error deleting buildRun.", namespace, request.Namespace, name, &buildRunSucceeded[i].Name, deleteError, deleteBuildRunErr)
					return reconcile.Result{}, nil
				}
			}
		}
	}

	if b.Spec.Retention.FailedLimit != nil {
		if len(buildRunFailed) > int(*b.Spec.Retention.FailedLimit) {
			sort.Slice(buildRunFailed, func(i, j int) bool {
				return buildRunFailed[i].Status.CompletionTime.Before(buildRunFailed[j].Status.CompletionTime)
			})
			lenOfList := len(buildRunFailed)
			for i := 0; lenOfList-i > int(*b.Spec.Retention.FailedLimit); i += 1 {
				ctxlog.Info(ctx, "Deleting failed buildrun as cleanup limit has been reached.", namespace, request.Namespace, name, buildRunFailed[i].Name)
				deleteBuildRunErr := r.client.Delete(ctx, &buildRunFailed[i], &client.DeleteOptions{})
				if deleteBuildRunErr != nil {
					if apierrors.IsNotFound(deleteBuildRunErr) {
						return reconcile.Result{}, nil
					}
					ctxlog.Debug(ctx, "Error deleting buildRun.", namespace, request.Namespace, name, buildRunFailed[i].Name, deleteError, deleteBuildRunErr)
					return reconcile.Result{}, nil
				}
			}
		}
	}

	ctxlog.Debug(ctx, "finishing reconciling request from a Build or BuildRun event", namespace, request.Namespace, name, request.Name)

	return reconcile.Result{}, nil
}

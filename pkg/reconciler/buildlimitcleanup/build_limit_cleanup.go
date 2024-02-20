// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildlimitcleanup

import (
	"context"
	"sort"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileBuild reconciles a Build object
type ReconcileBuild struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver */
	config *config.Config
	client client.Client
}

func NewReconciler(c *config.Config, mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileBuild{
		config: c,
		client: mgr.GetClient(),
	}
}

// Reconciler finds retentions fields in builds and makes sure the
// number of corresponding buildruns adhere to these limits
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

	// early exit if there is no retention section
	if b.Spec.Retention == nil {
		return reconcile.Result{}, nil
	}

	// early exit if retention section has no limit
	if b.Spec.Retention.SucceededLimit == nil && b.Spec.Retention.FailedLimit == nil {
		return reconcile.Result{}, nil
	}

	lbls := map[string]string{
		build.LabelBuild: b.Name,
	}
	opts := client.ListOptions{
		Namespace:     b.Namespace,
		LabelSelector: labels.SelectorFromSet(lbls),
	}
	allBuildRuns := &build.BuildRunList{}

	err = r.client.List(ctx, allBuildRuns, &opts)
	if err != nil {
		return reconcile.Result{}, err
	}

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
			// Sort buildruns with oldest one at the beginning
			sort.Slice(buildRunSucceeded, func(i, j int) bool {
				return buildRunSucceeded[i].ObjectMeta.CreationTimestamp.Before(&buildRunSucceeded[j].ObjectMeta.CreationTimestamp)
			})
			lenOfList := len(buildRunSucceeded)
			for i := 0; i < lenOfList-int(*b.Spec.Retention.SucceededLimit); i++ {
				ctxlog.Info(ctx, "Deleting succeeded buildrun as cleanup limit has been reached.", namespace, request.Namespace, name, buildRunSucceeded[i].Name)
				err := r.client.Delete(ctx, &buildRunSucceeded[i], &client.DeleteOptions{})
				if err != nil {
					if !apierrors.IsNotFound(err) {
						ctxlog.Debug(ctx, "Error deleting buildRun.", namespace, request.Namespace, name, &buildRunSucceeded[i].Name, "error", err)
						return reconcile.Result{}, err
					}
					ctxlog.Debug(ctx, "Error deleting buildRun. It has already been deleted.", namespace, request.Namespace, name, &buildRunSucceeded[i].Name)
				}
			}
		}
	}

	if b.Spec.Retention.FailedLimit != nil {
		if len(buildRunFailed) > int(*b.Spec.Retention.FailedLimit) {
			// Sort buildruns with oldest one at the beginning
			sort.Slice(buildRunFailed, func(i, j int) bool {
				return buildRunFailed[i].ObjectMeta.CreationTimestamp.Before(&buildRunFailed[j].ObjectMeta.CreationTimestamp)
			})
			lenOfList := len(buildRunFailed)
			for i := 0; i < lenOfList-int(*b.Spec.Retention.FailedLimit); i++ {
				ctxlog.Info(ctx, "Deleting failed buildrun as cleanup limit has been reached.", namespace, request.Namespace, name, buildRunFailed[i].Name)
				err := r.client.Delete(ctx, &buildRunFailed[i], &client.DeleteOptions{})
				if err != nil {
					if !apierrors.IsNotFound(err) {
						ctxlog.Debug(ctx, "Error deleting buildRun.", namespace, request.Namespace, name, &buildRunFailed[i].Name, "error", err)
						return reconcile.Result{}, err
					}
					ctxlog.Debug(ctx, "Error deleting buildRun. It has already been deleted.", namespace, request.Namespace, name, &buildRunFailed[i].Name)
				}
			}
		}
	}

	ctxlog.Debug(ctx, "finishing reconciling request from a Build or BuildRun event", namespace, request.Namespace, name, request.Name)

	return reconcile.Result{}, nil
}

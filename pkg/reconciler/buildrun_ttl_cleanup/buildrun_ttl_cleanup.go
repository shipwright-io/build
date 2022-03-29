// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun_ttl_cleanup

import (
	"context"
	"time"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileBuildRun reconciles a BuildRun object

type ReconcileBuildRun struct {
	/* This client, initialized using mgr.Client() above, is a split client
	   that reads objects from the cache and writes to the apiserver */
	config                *config.Config
	client                client.Client
	scheme                *runtime.Scheme
	setOwnerReferenceFunc setOwnerReferenceFunc
}

func NewReconciler(c *config.Config, mgr manager.Manager, ownerRef setOwnerReferenceFunc) reconcile.Reconciler {
	return &ReconcileBuildRun{
		config:                c,
		client:                mgr.GetClient(),
		scheme:                mgr.GetScheme(),
		setOwnerReferenceFunc: ownerRef,
	}
}

// GetBuildRunObject retrieves an existing BuildRun based on a name and namespace
func (r *ReconcileBuildRun) GetBuildRunObject(ctx context.Context, objectName string, objectNS string, buildRun *buildv1alpha1.BuildRun) error {
	if err := r.client.Get(ctx, types.NamespacedName{Name: objectName, Namespace: objectNS}, buildRun); err != nil {
		return err
	}
	return nil
}

/* Reconciler makes sure the buildrun adheres to its ttl retention field and deletes it
   once the ttl limit is hit */
func (r *ReconcileBuildRun) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// Set the ctx to be Background, as the top-level context for incoming requests.if
	ctx, cancel := context.WithTimeout(ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Debug(ctx, "Start reconciling Buildrun-ttl", namespace, request.Namespace, name, request.Name)

	br := &buildv1alpha1.BuildRun{}
	err := r.GetBuildRunObject(ctx, request.Name, request.Namespace, br)
	if err != nil {
		if apierrors.IsNotFound(err) {
			ctxlog.Debug(ctx, "Finish reconciling buildrun-ttl. Buildrun was not found", namespace, request.Namespace, name, request.Name)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	condition := br.Status.GetCondition(buildv1alpha1.Succeeded)
	if condition == nil {
		return reconcile.Result{}, nil
	}

	/* In case ttl has been reached, delete the buildrun, if not,
	   calculate the remaining time and requeue the buildrun */
	switch condition.Status {

	case corev1.ConditionTrue:
		if br.Status.BuildSpec.Retention.TtlAfterSucceeded != nil {
			if br.Status.CompletionTime.Add(br.Status.BuildSpec.Retention.TtlAfterSucceeded.Duration).Before(time.Now()) {
				ctxlog.Info(ctx, "Deleting successful buildrun as ttl has been reached.", namespace, request.Namespace, name, request.Name)
				deleteBuildRunErr := r.client.Delete(ctx, br, &client.DeleteOptions{})
				if deleteBuildRunErr != nil {
					if apierrors.IsNotFound(deleteBuildRunErr) {
						return reconcile.Result{}, nil
					}
					ctxlog.Debug(ctx, "Error deleting buildRun.", namespace, request.Namespace, name, br.Name, deleteError, deleteBuildRunErr)
					return reconcile.Result{}, deleteBuildRunErr
				}
			} else {
				timeLeft := br.Status.CompletionTime.Add(br.Status.BuildSpec.Retention.TtlAfterSucceeded.Duration).Sub(time.Now())
				return reconcile.Result{Requeue: true, RequeueAfter: timeLeft}, nil
			}
		}

	case corev1.ConditionFalse:
		if br.Status.BuildSpec.Retention.TtlAfterFailed != nil {
			if br.Status.CompletionTime.Add(br.Status.BuildSpec.Retention.TtlAfterFailed.Duration).Before(time.Now()) {
				ctxlog.Info(ctx, "Deleting failed buildrun as ttl has been reached.", namespace, request.Namespace, name, request.Name)
				deleteBuildRunErr := r.client.Delete(ctx, br, &client.DeleteOptions{})
				if deleteBuildRunErr != nil {
					if apierrors.IsNotFound(deleteBuildRunErr) {
						return reconcile.Result{}, nil
					}
					ctxlog.Debug(ctx, "Error deleting buildRun.", namespace, request.Namespace, name, br.Name, deleteError, deleteBuildRunErr)
					return reconcile.Result{}, deleteBuildRunErr
				}
			} else {
				timeLeft := br.Status.CompletionTime.Add(br.Status.BuildSpec.Retention.TtlAfterFailed.Duration).Sub(time.Now())
				return reconcile.Result{Requeue: true, RequeueAfter: timeLeft}, nil
			}
		}
	}
	ctxlog.Debug(ctx, "Finishing reconciling request from a BuildRun event", namespace, request.Namespace, name, request.Name)
	return reconcile.Result{}, nil
}

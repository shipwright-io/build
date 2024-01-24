// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrunttlcleanup

import (
	"context"
	"time"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileBuildRun reconciles a BuildRun object
type ReconcileBuildRun struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	config *config.Config
	client client.Client
}

func NewReconciler(c *config.Config, mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileBuildRun{
		config: c,
		client: mgr.GetClient(),
	}
}

// Reconcile makes sure the buildrun adheres to its ttl retention field and deletes it
// once the ttl limit is hit
func (r *ReconcileBuildRun) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Debug(ctx, "Start reconciling Buildrun-ttl", namespace, request.Namespace, name, request.Name)

	br := &buildv1beta1.BuildRun{}
	err := r.client.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, br)
	if err != nil {
		if apierrors.IsNotFound(err) {
			ctxlog.Debug(ctx, "Finish reconciling buildrun-ttl. Buildrun was not found", namespace, request.Namespace, name, request.Name)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	condition := br.Status.GetCondition(buildv1beta1.Succeeded)
	if condition == nil || condition.Status == corev1.ConditionUnknown {
		return reconcile.Result{}, nil
	}
	var ttl *metav1.Duration
	if condition.Status == corev1.ConditionTrue {
		if br.Spec.Retention != nil && br.Spec.Retention.TTLAfterSucceeded != nil {
			ttl = br.Spec.Retention.TTLAfterSucceeded
		} else if br.Status.BuildSpec != nil && br.Status.BuildSpec.Retention != nil && br.Status.BuildSpec.Retention.TTLAfterSucceeded != nil {
			ttl = br.Status.BuildSpec.Retention.TTLAfterSucceeded
		}
	} else {
		if br.Spec.Retention != nil && br.Spec.Retention.TTLAfterFailed != nil {
			ttl = br.Spec.Retention.TTLAfterFailed
		} else if br.Status.BuildSpec != nil && br.Status.BuildSpec.Retention != nil && br.Status.BuildSpec.Retention.TTLAfterFailed != nil {
			ttl = br.Status.BuildSpec.Retention.TTLAfterFailed
		}
	}

	// check if BuildRun still has a TTL
	if ttl == nil {
		return reconcile.Result{}, nil
	}

	if br.Status.CompletionTime.Add(ttl.Duration).Before(time.Now()) {
		ctxlog.Info(ctx, "Deleting buildrun as ttl has been reached.", namespace, request.Namespace, name, request.Name)
		err := r.client.Delete(ctx, br, &client.DeleteOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				ctxlog.Debug(ctx, "Error deleting buildRun.", namespace, request.Namespace, name, br.Name, "error", err)
				return reconcile.Result{}, err
			}
			ctxlog.Debug(ctx, "Error deleting buildRun. It has already been deleted.", namespace, request.Namespace, name, br.Name)
			return reconcile.Result{}, nil
		}
	} else {
		timeLeft := time.Until(br.Status.CompletionTime.Add(ttl.Duration))
		return reconcile.Result{Requeue: true, RequeueAfter: timeLeft}, nil
	}

	ctxlog.Debug(ctx, "Finishing reconciling request from a BuildRun event", namespace, request.Namespace, name, request.Name)
	return reconcile.Result{}, nil
}

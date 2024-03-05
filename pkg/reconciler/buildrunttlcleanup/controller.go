// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrunttlcleanup

import (
	"context"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	namespace string = "namespace"
	name      string = "name"
)

// Add creates a new BuildRun_ttl_cleanup Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(_ context.Context, c *config.Config, mgr manager.Manager) error {
	return add(mgr, NewReconciler(c, mgr), c.Controllers.BuildRun.MaxConcurrentReconciles)
}

// reconcileCompletedBuildRun returns true if the object has the required TTL parameters
func reconcileCompletedBuildRun(condition *buildv1beta1.Condition, o *buildv1beta1.BuildRun) bool {
	if condition.Status == corev1.ConditionTrue {
		// check if a successful BuildRun has a TTL after succeeded value set
		if o.Spec.Retention != nil && o.Spec.Retention.TTLAfterSucceeded != nil {
			return true
		}

		if o.Status.BuildSpec != nil && o.Status.BuildSpec.Retention != nil && o.Status.BuildSpec.Retention.TTLAfterSucceeded != nil {
			return true
		}
	} else {
		// check if a failed BuildRun has a TTL after failed
		if o.Spec.Retention != nil && o.Spec.Retention.TTLAfterFailed != nil {
			return true
		}

		if o.Status.BuildSpec != nil && o.Status.BuildSpec.Retention != nil && o.Status.BuildSpec.Retention.TTLAfterFailed != nil {
			return true
		}
	}
	return false
}

// reconcileAlreadyCompletedBuildRun returns true only if the TTL limit was introduced
// or if it was lowered as the object was completed before the update
func reconcileAlreadyCompletedBuildRun(newCondition *buildv1beta1.Condition, n *buildv1beta1.BuildRun, o *buildv1beta1.BuildRun) bool {
	if newCondition.Status == corev1.ConditionTrue {
		// check if a successful BuildRun has a TTL that was lowered or introduced
		if (o.Spec.Retention == nil || o.Spec.Retention.TTLAfterSucceeded == nil) && n.Spec.Retention != nil && n.Spec.Retention.TTLAfterSucceeded != nil {
			return true
		}

		if o.Spec.Retention != nil && o.Spec.Retention.TTLAfterSucceeded != nil && n.Spec.Retention != nil && n.Spec.Retention.TTLAfterSucceeded != nil && n.Spec.Retention.TTLAfterSucceeded.Duration < o.Spec.Retention.TTLAfterSucceeded.Duration {
			return true
		}
	} else {
		// check if a failed BuildRun has a TTL that was lowered or introduced
		if (o.Spec.Retention == nil || o.Spec.Retention.TTLAfterFailed == nil) && n.Spec.Retention != nil && n.Spec.Retention.TTLAfterFailed != nil {
			return true
		}

		if o.Spec.Retention != nil && o.Spec.Retention.TTLAfterFailed != nil && n.Spec.Retention != nil && n.Spec.Retention.TTLAfterFailed != nil && n.Spec.Retention.TTLAfterFailed.Duration < o.Spec.Retention.TTLAfterFailed.Duration {
			return true
		}
	}
	return false
}

func add(mgr manager.Manager, r reconcile.Reconciler, maxConcurrentReconciles int) error {
	// Create the controller options
	options := controller.Options{
		Reconciler: r,
	}

	if maxConcurrentReconciles > 0 {
		options.MaxConcurrentReconciles = maxConcurrentReconciles

	}

	c, err := controller.New("buildrun-ttl-cleanup-controller", mgr, options)
	if err != nil {
		return err
	}

	predBuildRun := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// Reconcile if TTL values are set
			o := e.Object.(*buildv1beta1.BuildRun)

			// ignore a running BuildRun
			condition := o.Status.GetCondition(buildv1beta1.Succeeded)
			if condition == nil || condition.Status == corev1.ConditionUnknown {
				return false
			}

			return reconcileCompletedBuildRun(condition, o)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// check if the updated object is completed
			n := e.ObjectNew.(*buildv1beta1.BuildRun)
			newCondition := n.Status.GetCondition(buildv1beta1.Succeeded)
			if newCondition == nil || newCondition.Status == corev1.ConditionUnknown {
				return false
			}

			o := e.ObjectOld.(*buildv1beta1.BuildRun)
			oldCondition := o.Status.GetCondition(buildv1beta1.Succeeded)

			// for objects that failed or just completed, check if a matching TTL is set
			if oldCondition == nil || oldCondition.Status == corev1.ConditionUnknown {
				return reconcileCompletedBuildRun(newCondition, n)
			}

			// for objects that were already complete, check if the TTL was lowered or introduced
			if oldCondition != nil && oldCondition.Status != corev1.ConditionUnknown {
				return reconcileAlreadyCompletedBuildRun(newCondition, n, o)
			}

			return false
		},
		DeleteFunc: func(_ event.DeleteEvent) bool {
			// Never reconcile on deletion, there is nothing we have to do
			return false
		},
	}
	// Watch for changes to primary resource BuildRun
	return c.Watch(source.Kind(mgr.GetCache(), &buildv1beta1.BuildRun{}), &handler.EnqueueRequestForObject{}, predBuildRun)
}

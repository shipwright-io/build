// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildlimitcleanup

import (
	"context"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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

// Add creates a new build_limit_cleanup Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started
func Add(_ context.Context, c *config.Config, mgr manager.Manager) error {
	return add(mgr, NewReconciler(c, mgr), c.Controllers.Build.MaxConcurrentReconciles)
}
func add(mgr manager.Manager, r reconcile.Reconciler, maxConcurrentReconciles int) error {
	// Create the controller options
	options := controller.Options{
		Reconciler: r,
	}

	if maxConcurrentReconciles > 0 {
		options.MaxConcurrentReconciles = maxConcurrentReconciles
	}

	// Create a new controller
	c, err := controller.New("build-limit-cleanup-controller", mgr, options)
	if err != nil {
		return err
	}

	pred := predicate.TypedFuncs[*buildv1beta1.Build]{
		CreateFunc: func(e event.TypedCreateEvent[*buildv1beta1.Build]) bool {
			o := e.Object
			return o.Spec.Retention != nil && (o.Spec.Retention.FailedLimit != nil || o.Spec.Retention.SucceededLimit != nil)
		},
		UpdateFunc: func(e event.TypedUpdateEvent[*buildv1beta1.Build]) bool {
			n := e.ObjectNew
			o := e.ObjectOld

			// Check to see if there are new retention parameters or whether the
			// limit values have decreased
			if o.Spec.Retention == nil && n.Spec.Retention != nil {
				if n.Spec.Retention.FailedLimit != nil || n.Spec.Retention.SucceededLimit != nil {
					return true
				}
			} else if n.Spec.Retention != nil && o.Spec.Retention != nil {
				if n.Spec.Retention.FailedLimit != nil && o.Spec.Retention.FailedLimit == nil {
					return true
				} else if n.Spec.Retention.SucceededLimit != nil && o.Spec.Retention.SucceededLimit == nil {
					return true
					// #nosec G115, is validated in the type
				} else if n.Spec.Retention.FailedLimit != nil && o.Spec.Retention.FailedLimit != nil && int(*n.Spec.Retention.FailedLimit) < int(*o.Spec.Retention.FailedLimit) {
					return true
					// #nosec G115, is validated in the type
				} else if n.Spec.Retention.SucceededLimit != nil && o.Spec.Retention.SucceededLimit != nil && int(*n.Spec.Retention.SucceededLimit) < int(*o.Spec.Retention.SucceededLimit) {
					return true
				}
			}
			return false
		},
		DeleteFunc: func(_ event.TypedDeleteEvent[*buildv1beta1.Build]) bool {
			// Never reconcile on deletion, there is nothing we have to do
			return false
		},
	}

	predBuildRun := predicate.TypedFuncs[*buildv1beta1.BuildRun]{
		CreateFunc: func(_ event.TypedCreateEvent[*buildv1beta1.BuildRun]) bool {
			// Never reconcile in case of create buildrun event
			return false
		},
		// Reconcile the build the related buildrun has just completed
		UpdateFunc: func(e event.TypedUpdateEvent[*buildv1beta1.BuildRun]) bool {
			n := e.ObjectNew

			// check if Buildrun is related to a build
			if n.Spec.Build.Name == nil {
				return false
			}

			o := e.ObjectOld
			oldCondition := o.Status.GetCondition(buildv1beta1.Succeeded)
			newCondition := n.Status.GetCondition(buildv1beta1.Succeeded)
			if newCondition != nil {
				if (oldCondition == nil || oldCondition.Status == corev1.ConditionUnknown) &&
					(newCondition.Status == corev1.ConditionFalse || newCondition.Status == corev1.ConditionTrue) {
					return true
				}
			}
			return false
		},
		DeleteFunc: func(_ event.TypedDeleteEvent[*buildv1beta1.BuildRun]) bool {
			// Never reconcile on deletion, there is nothing we have to do
			return false
		},
	}

	// Watch for changes to primary resource Build
	if err = c.Watch(source.Kind(mgr.GetCache(), &buildv1beta1.Build{}, &handler.TypedEnqueueRequestForObject[*buildv1beta1.Build]{}, pred)); err != nil {
		return err
	}

	// Watch for changes to resource BuildRun
	return c.Watch(source.Kind(mgr.GetCache(), &buildv1beta1.BuildRun{}, handler.TypedEnqueueRequestsFromMapFunc(func(_ context.Context, buildRun *buildv1beta1.BuildRun) []reconcile.Request {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      *buildRun.Spec.Build.Name,
					Namespace: buildRun.Namespace,
				},
			},
		}
	}), predBuildRun))
}

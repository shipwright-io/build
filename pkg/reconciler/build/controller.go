// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"context"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
)

const (
	namespace string = "namespace"
	name      string = "name"
)

type setOwnerReferenceFunc func(owner, object metav1.Object, scheme *runtime.Scheme, opts ...controllerutil.OwnerReferenceOption) error

// Add creates a new Build Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(ctx context.Context, c *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContext(ctx, "build-controller")
	return add(ctx, mgr, NewReconciler(c, mgr, controllerutil.SetControllerReference), c.Controllers.Build.MaxConcurrentReconciles)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(ctx context.Context, mgr manager.Manager, r reconcile.Reconciler, maxConcurrentReconciles int) error {
	// Create the controller options
	options := controller.Options{
		Reconciler: r,
	}
	if maxConcurrentReconciles > 0 {
		options.MaxConcurrentReconciles = maxConcurrentReconciles
	}

	// Create a new controller
	c, err := controller.New("build-controller", mgr, options)
	if err != nil {
		return err
	}

	pred := predicate.TypedFuncs[*buildapi.Build]{
		UpdateFunc: func(e event.TypedUpdateEvent[*buildapi.Build]) bool {
			o := e.ObjectOld
			n := e.ObjectNew

			buildAtBuildDeletion := false

			// Check if the Build retention AtBuildDeletion is updated
			oldBuildRetention := o.Spec.Retention
			newBuildRetention := n.Spec.Retention

			logAndEnableDeletion := func() {
				ctxlog.Debug(
					ctx,
					"updating predicated passed, the build retention AtBuildDeletion was modified.",
					namespace,
					n.GetNamespace(),
					name,
					n.GetName(),
				)
				buildAtBuildDeletion = true
			}

			xorBuildRetentions := func(oldDeletion, newDeletion *bool) bool {
				if oldDeletion == nil {
					oldDeletion = ptr.To(false)
				}
				if newDeletion == nil {
					newDeletion = ptr.To(false)
				}
				return *oldDeletion != *newDeletion
			}

			if !reflect.DeepEqual(oldBuildRetention, newBuildRetention) {
				switch {
				case o.Spec.Retention == nil && n.Spec.Retention != nil:
					if n.Spec.Retention.AtBuildDeletion != nil && *n.Spec.Retention.AtBuildDeletion {
						logAndEnableDeletion()
					}
				case o.Spec.Retention != nil && n.Spec.Retention == nil:
					if o.Spec.Retention.AtBuildDeletion != nil && *o.Spec.Retention.AtBuildDeletion {
						logAndEnableDeletion()
					}
				case o.Spec.Retention != nil && n.Spec.Retention != nil:
					if xorBuildRetentions(o.Spec.Retention.AtBuildDeletion, n.Spec.Retention.AtBuildDeletion) {
						logAndEnableDeletion()
					}
				}
			}

			// Ignore updates to CR status in which case metadata.Generation does not change
			// or BuildRunDeletion annotation does not change
			return o.GetGeneration() != n.GetGeneration() || buildAtBuildDeletion
		},
		DeleteFunc: func(_ event.TypedDeleteEvent[*buildapi.Build]) bool {
			// Never reconcile on deletion, there is nothing we have to do
			return false
		},
	}

	// Watch for changes to primary resource Build
	if err = c.Watch(source.Kind(mgr.GetCache(), &buildapi.Build{}, &handler.TypedEnqueueRequestForObject[*buildapi.Build]{}, pred)); err != nil {
		return err
	}

	return c.Watch(source.Kind(mgr.GetCache(), &corev1.Secret{}, handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, secret *corev1.Secret) []reconcile.Request {
		buildList := &buildapi.BuildList{}

		// List all builds in the namespace of the current secret
		if err := mgr.GetClient().List(ctx, buildList, &client.ListOptions{Namespace: secret.Namespace}); err != nil {
			// Avoid entering into the Reconcile space
			ctxlog.Info(ctx, "unexpected error happened while listing builds", namespace, secret.Namespace, "error", err)
			return []reconcile.Request{}
		}

		if len(buildList.Items) == 0 {
			// Avoid entering into the Reconcile space
			return []reconcile.Request{}
		}

		// Only enter the Reconcile space if the secret is referenced on
		// any Build in the same namespaces

		reconcileList := []reconcile.Request{}

		for _, build := range buildList.Items {
			// Check if this specific Build references the Secret in source or output
			if (build.GetSourceCredentials() != nil && *build.GetSourceCredentials() == secret.Name) ||
				(build.Spec.Output.PushSecret != nil && *build.Spec.Output.PushSecret == secret.Name) {

				reconcileList = append(reconcileList, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      build.Name,
						Namespace: build.Namespace,
					},
				})
			}
		}
		return reconcileList
	})))
}

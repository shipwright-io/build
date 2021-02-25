// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun

import (
	"context"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/apis"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
)

type setOwnerReferenceFunc func(owner, object metav1.Object, scheme *runtime.Scheme) error

// Add creates a new BuildRun Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(ctx context.Context, c *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContext(ctx, "buildrun-controller")
	return add(ctx, mgr, NewReconciler(ctx, c, mgr, controllerutil.SetControllerReference))
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(ctx context.Context, mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("buildrun-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	predBuildRun := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			o := e.Object.(*buildv1alpha1.BuildRun)

			// The CreateFunc is also called when the controller is started and iterates over all objects. For those BuildRuns that have a TaskRun referenced already,
			// we do not need to do a further reconciliation. BuildRun updates then only happen from the TaskRun.
			return o.Status.LatestTaskRunRef == nil
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			o := e.ObjectOld.(*buildv1alpha1.BuildRun)

			// Avoid reconciling when for updates on the BuildRun, the build.build.dev/name
			// label is set, and when a BuildRun already have a referenced TaskRun.
			if o.GetLabels()[buildv1alpha1.LabelBuild] == "" || o.Status.LatestTaskRunRef != nil {
				return false
			}

			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Never reconcile on deletion, there is nothing we have to do
			return false
		},
	}

	predTaskRun := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			o := e.ObjectOld.(*v1beta1.TaskRun)
			n := e.ObjectNew.(*v1beta1.TaskRun)

			// Process an update event when the old TR resource is not yet started and the new TR resource got a
			// condition of the type Succeeded
			if o.Status.StartTime.IsZero() && n.Status.GetCondition(apis.ConditionSucceeded) != nil {
				return true
			}

			// Process an update event for every change in the condition.Reason between the old and new TR resource
			if o.Status.GetCondition(apis.ConditionSucceeded) != nil && n.Status.GetCondition(apis.ConditionSucceeded) != nil {
				if o.Status.GetCondition(apis.ConditionSucceeded).Reason != n.Status.GetCondition(apis.ConditionSucceeded).Reason {
					return true
				}
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			o := e.Object.(*v1beta1.TaskRun)

			// If the TaskRun was deleted before completion, then we reconcile to update the BuildRun to a Failed status
			return o.Status.CompletionTime == nil
		},
	}

	// Watch for changes to primary resource BuildRun
	err = c.Watch(&source.Kind{Type: &buildv1alpha1.BuildRun{}}, &handler.EnqueueRequestForObject{}, predBuildRun)
	if err != nil {
		return err
	}

	// enqueue Reconciles requests only for events where a TaskRun already exists and that is related
	// to a BuildRun
	return c.Watch(&source.Kind{Type: &v1beta1.TaskRun{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(o handler.MapObject) []reconcile.Request {

			taskRun := o.Object.(*v1beta1.TaskRun)

			// check if TaskRun is related to BuildRun
			if taskRun.GetLabels() == nil || taskRun.GetLabels()[buildv1alpha1.LabelBuildRun] == "" {
				return []reconcile.Request{}
			}

			return []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      taskRun.Name,
						Namespace: taskRun.Namespace,
					},
				},
			}
		}),
	}, predTaskRun)
}

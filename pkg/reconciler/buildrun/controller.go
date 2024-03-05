// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun

import (
	"context"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"knative.dev/pkg/apis"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
)

type setOwnerReferenceFunc func(owner, object metav1.Object, scheme *runtime.Scheme) error

// Add creates a new BuildRun Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(_ context.Context, c *config.Config, mgr manager.Manager) error {
	return add(mgr, NewReconciler(c, mgr, controllerutil.SetControllerReference), c.Controllers.BuildRun.MaxConcurrentReconciles)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler, maxConcurrentReconciles int) error {
	// Create the controller options
	options := controller.Options{
		Reconciler: r,
	}
	if maxConcurrentReconciles > 0 {
		options.MaxConcurrentReconciles = maxConcurrentReconciles
	}

	// Create a new controller
	c, err := controller.New("buildrun-controller", mgr, options)
	if err != nil {
		return err
	}

	predBuildRun := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			o := e.Object.(*buildv1beta1.BuildRun)

			// The CreateFunc is also called when the controller is started and iterates over all objects. For those BuildRuns that have a TaskRun referenced already,
			// we do not need to do a further reconciliation. BuildRun updates then only happen from the TaskRun.
			return o.Status.TaskRunName == nil && o.Status.CompletionTime == nil
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			o := e.ObjectOld.(*buildv1beta1.BuildRun)
			n := e.ObjectNew.(*buildv1beta1.BuildRun)

			// Only reconcile a BuildRun update when
			// - it is set to canceled
			switch {
			case !o.IsCanceled() && n.IsCanceled():
				return true
			}

			return false
		},
		DeleteFunc: func(_ event.DeleteEvent) bool {
			// Never reconcile on deletion, there is nothing we have to do
			return false
		},
	}

	predTaskRun := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			o := e.ObjectOld.(*pipelineapi.TaskRun)
			n := e.ObjectNew.(*pipelineapi.TaskRun)

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
			o := e.Object.(*pipelineapi.TaskRun)

			// If the TaskRun was deleted before completion, then we reconcile to update the BuildRun to a Failed status
			return o.Status.CompletionTime == nil
		},
	}

	// Watch for changes to primary resource BuildRun
	if err = c.Watch(source.Kind(mgr.GetCache(), &buildv1beta1.BuildRun{}), &handler.EnqueueRequestForObject{}, predBuildRun); err != nil {
		return err
	}

	// enqueue Reconciles requests only for events where a TaskRun already exists and that is related
	// to a BuildRun
	return c.Watch(source.Kind(mgr.GetCache(), &pipelineapi.TaskRun{}), handler.EnqueueRequestsFromMapFunc(func(_ context.Context, o client.Object) []reconcile.Request {
		taskRun := o.(*pipelineapi.TaskRun)

		// check if TaskRun is related to BuildRun
		if taskRun.GetLabels() == nil || taskRun.GetLabels()[buildv1beta1.LabelBuildRun] == "" {
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
	}), predTaskRun)
}

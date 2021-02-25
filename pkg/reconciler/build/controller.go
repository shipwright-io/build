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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
)

const (
	namespace string = "namespace"
	name      string = "name"
)

type setOwnerReferenceFunc func(owner, object metav1.Object, scheme *runtime.Scheme) error

// Add creates a new Build Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(ctx context.Context, c *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContext(ctx, "build-controller")
	return add(ctx, mgr, NewReconciler(ctx, c, mgr, controllerutil.SetControllerReference))
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(ctx context.Context, mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("build-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			buildRunDeletionAnnotation := false
			// Check if the AnnotationBuildRunDeletion annotation is updated
			oldAnnot := e.MetaOld.GetAnnotations()
			newAnnot := e.MetaNew.GetAnnotations()
			if !reflect.DeepEqual(oldAnnot, newAnnot) {
				if oldAnnot[build.AnnotationBuildRunDeletion] != newAnnot[build.AnnotationBuildRunDeletion] {
					ctxlog.Debug(
						ctx,
						"updating predicated passed, the annotation was modified.",
						namespace,
						e.MetaNew.GetNamespace(),
						name,
						e.MetaNew.GetName(),
					)
					buildRunDeletionAnnotation = true
				}
			}

			// Ignore updates to CR status in which case metadata.Generation does not change
			// or BuildRunDeletion annotation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration() || buildRunDeletionAnnotation
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Never reconcile on deletion, there is nothing we have to do
			return false
		},
	}

	// Watch for changes to primary resource Build
	err = c.Watch(&source.Kind{Type: &build.Build{}}, &handler.EnqueueRequestForObject{}, pred)
	if err != nil {
		return err
	}

	preSecret := predicate.Funcs{
		// Only filter events where the secret have the Build specific annotation
		CreateFunc: func(e event.CreateEvent) bool {
			objectAnnotations := e.Meta.GetAnnotations()
			if _, ok := buildSecretRefAnnotationExist(objectAnnotations); ok {
				return true
			}
			return false
		},

		// Only filter events where the secret have the Build specific annotation,
		// but only if the Build specific annotation changed
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldAnnotations := e.MetaOld.GetAnnotations()
			newAnnotations := e.MetaNew.GetAnnotations()

			if _, oldBuildKey := buildSecretRefAnnotationExist(oldAnnotations); !oldBuildKey {
				if _, newBuildKey := buildSecretRefAnnotationExist(newAnnotations); newBuildKey {
					return true
				}
			}
			return false
		},

		// Only filter events where the secret have the Build specific annotation
		DeleteFunc: func(e event.DeleteEvent) bool {
			objectAnnotations := e.Meta.GetAnnotations()
			if _, ok := buildSecretRefAnnotationExist(objectAnnotations); ok {
				return true
			}
			return false
		},
	}

	return c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(o handler.MapObject) []reconcile.Request {

			secret := o.Object.(*corev1.Secret)

			buildList := &build.BuildList{}

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
			flagReconcile := false

			for _, build := range buildList.Items {
				if build.Spec.Source.SecretRef != nil {
					if build.Spec.Source.SecretRef.Name == secret.Name {
						flagReconcile = true
					}
				}
				if build.Spec.Output.SecretRef != nil {
					if build.Spec.Output.SecretRef.Name == secret.Name {
						flagReconcile = true
					}
				}
				if build.Spec.BuilderImage != nil && build.Spec.BuilderImage.SecretRef != nil {
					if build.Spec.BuilderImage.SecretRef.Name == secret.Name {
						flagReconcile = true
					}
				}
				if flagReconcile {
					reconcileList = append(reconcileList, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      build.Name,
							Namespace: build.Namespace,
						},
					})
				}
			}
			return reconcileList
		}),
	}, preSecret)
}

func buildSecretRefAnnotationExist(annotation map[string]string) (string, bool) {
	if val, ok := annotation[build.AnnotationBuildRefSecret]; ok {
		return val, true
	}
	return "", false
}

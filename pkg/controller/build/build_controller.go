// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"context"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	buildmetrics "github.com/shipwright-io/build/pkg/metrics"
	"github.com/shipwright-io/build/pkg/validate"
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

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(ctx context.Context, c *config.Config, mgr manager.Manager, ownerRef setOwnerReferenceFunc) reconcile.Reconciler {
	return &ReconcileBuild{
		ctx:                   ctx,
		config:                c,
		client:                mgr.GetClient(),
		scheme:                mgr.GetScheme(),
		setOwnerReferenceFunc: ownerRef,
	}
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
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
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

// blank assignment to verify that ReconcileBuild implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileBuild{}

// ReconcileBuild reconciles a Build object
type ReconcileBuild struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	ctx                   context.Context
	config                *config.Config
	client                client.Client
	scheme                *runtime.Scheme
	setOwnerReferenceFunc setOwnerReferenceFunc
}

// Reconcile reads that state of the cluster for a Build object and makes changes based on the state read
// and what is in the Build.Spec
func (r *ReconcileBuild) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(r.ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Debug(ctx, "start reconciling Build", namespace, request.Namespace, name, request.Name)

	b := &build.Build{}
	err := r.client.Get(ctx, request.NamespacedName, b)
	if err != nil && !apierrors.IsNotFound(err) {
		return reconcile.Result{}, err
	} else if apierrors.IsNotFound(err) {
		ctxlog.Debug(ctx, "finish reconciling build. build was not found", namespace, request.Namespace, name, request.Name)
		return reconcile.Result{}, nil
	}

	// Populate the status struct with default values
	b.Status.Registered = corev1.ConditionFalse
	b.Status.Reason = build.SucceedStatus

	// build a list of current validation types
	validationTypes := []string{
		validate.OwnerReferences,
		validate.SourceURL,
		validate.Secrets,
		validate.Strategies,
		validate.Runtime,
	}

	// trigger all current validations
	for _, validationType := range validationTypes {
		v, err := validate.NewValidation(validationType, b, r.client, r.scheme)
		if err != nil {
			// when the validation type is unknown
			return reconcile.Result{}, err
		}

		if err := v.ValidatePath(ctx); err != nil {
			// We enqueue another reconcile here. This is done only for validation
			// types where the error can be produced from a failed API call.
			if validationType == validate.Secrets || validationType == validate.Strategies {
				return reconcile.Result{}, err
			}
			if validationType == validate.OwnerReferences {
				// we do not want to bail out here if the owerreference validation fails, we ignore this error on purpose
				// In case we just created the Build, we want the Build reconcile logic to continue, in order to
				// validate the Build references ( e.g secrets, strategies )
				ctxlog.Info(ctx, "unexpected error during ownership reference validation", namespace, request.Namespace, name, request.Name, "error", err)
			}
		}
		if b.Status.Reason != build.SucceedStatus {
			return r.UpdateBuildStatusAndRetreat(ctx, b)
		}
	}

	b.Status.Registered = corev1.ConditionTrue
	b.Status.Message = build.AllValidationsSucceeded
	err = r.client.Status().Update(ctx, b)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Increase Build count in metrics
	buildmetrics.BuildCountInc(b.Spec.StrategyRef.Name, b.Namespace, b.Name)

	ctxlog.Debug(ctx, "finishing reconciling Build", namespace, request.Namespace, name, request.Name)
	return reconcile.Result{}, nil
}

// UpdateBuildStatusAndRetreat returns an error if an update fails, this should force
// a new reconcile until the API call succeeds. If return is nil, no further reconciliations
// will take place
func (r *ReconcileBuild) UpdateBuildStatusAndRetreat(ctx context.Context, b *build.Build) (reconcile.Result, error) {
	if err := r.client.Status().Update(ctx, b); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func buildSecretRefAnnotationExist(annotation map[string]string) (string, bool) {
	if val, ok := annotation[build.AnnotationBuildRefSecret]; ok {
		return val, true
	}
	return "", false
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/controller/utils"
	"github.com/shipwright-io/build/pkg/ctxlog"
	buildmetrics "github.com/shipwright-io/build/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// succeedStatus default status for the Build CRD
const succeedStatus string = "Succeeded"
const namespace string = "namespace"
const name string = "name"

// Add creates a new Build Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(ctx context.Context, c *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContext(ctx, "build-controller")
	return add(ctx, mgr, NewReconciler(ctx, c, mgr))
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(ctx context.Context, c *config.Config, mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileBuild{
		ctx:    ctx,
		config: c,
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
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
			buildRunFinalizer := false
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
					buildRunFinalizer = true
				}
			}

			// Ignore updates to CR status in which case metadata.Generation does not change
			// or BuildRunDeletion annotation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration() || buildRunFinalizer
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

	return nil
}

// blank assignment to verify that ReconcileBuild implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileBuild{}

// ReconcileBuild reconciles a Build object
type ReconcileBuild struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	ctx    context.Context
	config *config.Config
	client client.Client
	scheme *runtime.Scheme
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

	// Add finalizer for build
	if err := r.configFinalizer(ctx, b); err != nil {
		return reconcile.Result{}, err
	}
	// Check if the build is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if !b.DeletionTimestamp.IsZero() {
		ctxlog.Info(ctx, "build is marked for deletion", namespace, b.Namespace, name, b.Name)
		return reconcile.Result{}, r.finalizeBuildRun(ctx, b)
	}

	// Populate the status struct with default values
	b.Status.Registered = corev1.ConditionFalse
	b.Status.Reason = succeedStatus

	// Validate if the referenced secrets exist in the namespace
	var secretNames []string
	if b.Spec.Output.SecretRef != nil && b.Spec.Output.SecretRef.Name != "" {
		secretNames = append(secretNames, b.Spec.Output.SecretRef.Name)
	}
	if b.Spec.Source.SecretRef != nil && b.Spec.Source.SecretRef.Name != "" {
		secretNames = append(secretNames, b.Spec.Source.SecretRef.Name)
	}
	if b.Spec.BuilderImage != nil && b.Spec.BuilderImage.SecretRef != nil && b.Spec.BuilderImage.SecretRef.Name != "" {
		secretNames = append(secretNames, b.Spec.BuilderImage.SecretRef.Name)
	}

	if len(secretNames) > 0 {
		if err := r.validateSecrets(ctx, secretNames, b.Namespace); err != nil {
			b.Status.Reason = err.Error()
			updateErr := r.client.Status().Update(ctx, b)
			return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
		}
	}

	// Validate if the build strategy is defined
	if b.Spec.StrategyRef != nil {
		if err := r.validateStrategyRef(ctx, b.Spec.StrategyRef, b.Namespace); err != nil {
			b.Status.Reason = err.Error()
			updateErr := r.client.Status().Update(ctx, b)
			return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
		}
		ctxlog.Info(ctx, "buildStrategy found", namespace, b.Namespace, name, b.Name, "strategy", b.Spec.StrategyRef.Name)
	}

	// validate if "spec.runtime" attributes are valid
	if utils.IsRuntimeDefined(b) {
		if err := r.validateRuntime(b.Spec.Runtime); err != nil {
			ctxlog.Error(ctx, err, "failed validating runtime attributes", "Build", b.Name)
			b.Status.Reason = err.Error()
			updateErr := r.client.Status().Update(ctx, b)
			return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
		}
	}

	b.Status.Registered = corev1.ConditionTrue
	err = r.client.Status().Update(ctx, b)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Increase Build count in metrics
	buildmetrics.BuildCountInc(b.Spec.StrategyRef.Name)

	ctxlog.Debug(ctx, "finishing reconciling Build", namespace, request.Namespace, name, request.Name)
	return reconcile.Result{}, nil
}

func (r *ReconcileBuild) validateRuntime(runtime *build.Runtime) error {
	if len(runtime.Paths) == 0 {
		return fmt.Errorf("the property 'spec.runtime.paths' must not be empty")
	}
	return nil
}

func (r *ReconcileBuild) validateStrategyRef(ctx context.Context, s *build.StrategyRef, ns string) error {
	if s.Kind != nil {
		switch *s.Kind {
		case build.NamespacedBuildStrategyKind:
			if err := r.validateBuildStrategy(ctx, s.Name, ns); err != nil {
				return err
			}
		case build.ClusterBuildStrategyKind:
			if err := r.validateClusterBuildStrategy(ctx, s.Name); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown strategy %v", *s.Kind)
		}
	} else {
		ctxlog.Info(ctx, "buildStrategy kind is nil, use default NamespacedBuildStrategyKind")
		if err := r.validateBuildStrategy(ctx, s.Name, ns); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileBuild) validateBuildStrategy(ctx context.Context, n string, ns string) error {
	list := &build.BuildStrategyList{}

	if err := r.client.List(ctx, list, &client.ListOptions{Namespace: ns}); err != nil {
		return errors.Wrapf(err, "listing BuildStrategies in ns %s failed", ns)
	}

	if len(list.Items) == 0 {
		return errors.Errorf("none BuildStrategies found in namespace %s", ns)
	}

	if len(list.Items) > 0 {
		for _, s := range list.Items {
			if s.Name == n {
				return nil
			}
		}
		return fmt.Errorf("buildStrategy %s does not exist in namespace %s", n, ns)
	}
	return nil
}

func (r *ReconcileBuild) validateClusterBuildStrategy(ctx context.Context, n string) error {
	list := &build.ClusterBuildStrategyList{}

	if err := r.client.List(ctx, list); err != nil {
		return errors.Wrapf(err, "listing ClusterBuildStrategies failed")
	}

	if len(list.Items) == 0 {
		return errors.Errorf("none ClusterBuildStrategies found")
	}

	if len(list.Items) > 0 {
		for _, s := range list.Items {
			if s.Name == n {
				return nil
			}
		}
		return fmt.Errorf("clusterBuildStrategy %s does not exist", n)
	}
	return nil
}

func (r *ReconcileBuild) validateSecrets(ctx context.Context, secretNames []string, ns string) error {
	list := &corev1.SecretList{}

	if err := r.client.List(
		ctx,
		list,
		&client.ListOptions{
			Namespace: ns,
		},
	); err != nil {
		return errors.Wrapf(err, "listing secrets in namespace %s failed", ns)
	}

	if len(list.Items) == 0 {
		return errors.Errorf("there are no secrets in namespace %s", ns)
	}

	var lookUp = map[string]bool{}
	for _, secretName := range secretNames {
		lookUp[secretName] = false
	}
	for _, secret := range list.Items {
		lookUp[secret.Name] = true
	}
	var missingSecrets []string
	for name, found := range lookUp {
		if !found {
			missingSecrets = append(missingSecrets, name)
		}
	}

	if len(missingSecrets) > 1 {
		return fmt.Errorf("secrets %s do not exist", strings.Join(missingSecrets, ", "))
	} else if len(missingSecrets) > 0 {
		return fmt.Errorf("secret %s does not exist", missingSecrets[0])
	}

	return nil
}

func (r *ReconcileBuild) configFinalizer(ctx context.Context, b *build.Build) error {
	if b.GetAnnotations()[build.AnnotationBuildRunDeletion] == "true" {
		if !contains(b.GetFinalizers(), build.BuildFinalizer) {
			ctxlog.Info(ctx, "add finalizer to build", namespace, b.Namespace, name, b.Name)
			b.SetFinalizers(append(b.GetFinalizers(), build.BuildFinalizer))
			if err := r.client.Update(ctx, b); err != nil {
				return err
			}
		}
	} else {
		if contains(b.GetFinalizers(), build.BuildFinalizer) {
			ctxlog.Info(ctx, "remove finalizer from build", namespace, b.Namespace, name, b.Name)
			b.SetFinalizers(remove(b.GetFinalizers(), build.BuildFinalizer))
			if err := r.client.Update(ctx, b); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *ReconcileBuild) finalizeBuildRun(ctx context.Context, b *build.Build) error {
	if contains(b.GetFinalizers(), build.BuildFinalizer) {
		// Run finalization logic for buildFinalizer. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if err := r.cleanBuildRun(ctx, b); err != nil {
			return err
		}

		// Remove buildFinalizer. Once all finalizers have been
		// removed, the object will be deleted.
		b.SetFinalizers(remove(b.GetFinalizers(), build.BuildFinalizer))
		err := r.client.Update(ctx, b)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileBuild) cleanBuildRun(ctx context.Context, b *build.Build) error {
	buildRunList := &build.BuildRunList{}

	lbls := map[string]string{
		build.LabelBuild: b.Name,
	}
	opts := client.ListOptions{
		Namespace:     b.Namespace,
		LabelSelector: labels.SelectorFromSet(lbls),
	}
	if err := r.client.List(ctx, buildRunList, &opts); err != nil {
		return err
	}

	for _, buildRun := range buildRunList.Items {
		ctxlog.Info(ctx, "deleting buildrun", namespace, b.Namespace, name, b.Name, "buildrunname", buildRun.Name)
		if err := r.client.Delete(ctx, &buildRun); err != nil {
			return err
		}
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}

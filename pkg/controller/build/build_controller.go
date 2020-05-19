package build

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	build "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
)

var log = logf.Log.WithName("controller_build")

// succeedStatus default status for the Build CRD
const succeedStatus string = "Succeeded"

// Add creates a new Build Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, NewReconciler(mgr))
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileBuild{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
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
	err = c.Watch(&source.Kind{Type: &buildv1alpha1.Build{}}, &handler.EnqueueRequestForObject{}, pred)
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
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Build object and makes changes based on the state read
// and what is in the Build.Spec
func (r *ReconcileBuild) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Build")

	b := &build.Build{}
	err := r.client.Get(context.Background(), request.NamespacedName, b)
	if err != nil && !apierrors.IsNotFound(err) {
		return reconcile.Result{}, err
	} else if apierrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}

	// Add finalizer for build
	if err := r.configFinalizer(b); err != nil {
		return reconcile.Result{}, err
	}
	// Check if the build is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if !b.DeletionTimestamp.IsZero() {
		finishClean, err := r.finalizeBuildRun(b)
		if err != nil {
			return reconcile.Result{}, err
		}
		if finishClean {
			return reconcile.Result{}, nil
		}
	}

	// Populate the status struct with default values
	b.Status.Registered = corev1.ConditionFalse
	b.Status.Reason = succeedStatus

	// Validate if the spec.output.secretref exist in the namespace
	if b.Spec.Output.SecretRef != nil && b.Spec.Output.SecretRef.Name != "" {
		if err := r.validateOutputSecret(b.Spec.Output.SecretRef.Name, b.Namespace); err != nil {
			reqLogger.Error(err, "failed validating the output secret", "Build", b.Name)
			b.Status.Reason = err.Error()
			updateErr := r.client.Status().Update(context.Background(), b)
			return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
		}
	}

	// Validate if the build strategy is defined
	if b.Spec.StrategyRef != nil {
		if err := r.validateStrategyRef(b.Spec.StrategyRef, b.Namespace); err != nil {
			reqLogger.Error(err, "failed validating the strategy reference", "Build", b.Name)
			b.Status.Reason = err.Error()
			updateErr := r.client.Status().Update(context.Background(), b)
			return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
		}
	}

	b.Status.Registered = corev1.ConditionTrue
	err = r.client.Status().Update(context.Background(), b)
	if err != nil {
		reqLogger.Error(err, "Failed to update the Build status", "Build", b.Name)
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileBuild) validateStrategyRef(s *build.StrategyRef, ns string) error {
	if s.Kind != nil {
		switch *s.Kind {
		case build.NamespacedBuildStrategyKind:
			if err := r.validateBuildStrategy(s.Name, ns); err != nil {
				return err
			}
		case build.ClusterBuildStrategyKind:
			if err := r.validateClusterBuildStrategy(s.Name); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown strategy %v", *s.Kind)
		}
	} else {
		log.Info("BuildStrategy kind is nil, use default NamespacedBuildStrategyKind")
		if err := r.validateBuildStrategy(s.Name, ns); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileBuild) validateBuildStrategy(n string, ns string) error {
	list := &build.BuildStrategyList{}

	if err := r.client.List(context.Background(), list, &client.ListOptions{Namespace: ns}); err != nil {
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
		return fmt.Errorf("BuildStrategy %s does not exist in namespace %s", n, ns)
	}
	return nil
}

func (r *ReconcileBuild) validateClusterBuildStrategy(n string) error {
	list := &build.ClusterBuildStrategyList{}

	if err := r.client.List(context.Background(), list); err != nil {
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

func (r *ReconcileBuild) validateOutputSecret(n string, ns string) error {
	list := &corev1.SecretList{}

	if err := r.client.List(
		context.Background(),
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

	if len(list.Items) > 0 {
		for _, secret := range list.Items {
			if secret.Name == n {
				return nil
			}
		}
		return fmt.Errorf("secret %s does not exist", n)
	}
	return nil
}

func (r *ReconcileBuild) configFinalizer(b *build.Build) error {
	if b.GetAnnotations()[build.AnnotationBuildRunDeletion] == "true" {
		if !contains(b.GetFinalizers(), build.BuildFinalizer) {
			b.SetFinalizers(append(b.GetFinalizers(), build.BuildFinalizer))
			if err := r.client.Update(context.TODO(), b); err != nil {
				return err
			}
		}
	} else {
		if contains(b.GetFinalizers(), build.BuildFinalizer) {
			b.SetFinalizers(remove(b.GetFinalizers(), build.BuildFinalizer))
			if err := r.client.Update(context.TODO(), b); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *ReconcileBuild) finalizeBuildRun(b *build.Build) (bool, error) {
	if contains(b.GetFinalizers(), build.BuildFinalizer) {
		// Run finalization logic for buildFinalizer. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if err := r.cleanBuildRun(b); err != nil {
			return false, err
		}

		// Remove buildFinalizer. Once all finalizers have been
		// removed, the object will be deleted.
		b.SetFinalizers(remove(b.GetFinalizers(), build.BuildFinalizer))
		err := r.client.Update(context.TODO(), b)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (r *ReconcileBuild) cleanBuildRun(b *build.Build) error {
	buildRunList := &build.BuildRunList{}

	lbls := map[string]string{
		buildv1alpha1.LabelBuild: b.Name,
	}
	opts := client.ListOptions{
		Namespace:     b.Namespace,
		LabelSelector: labels.SelectorFromSet(lbls),
	}
	if err := r.client.List(context.TODO(), buildRunList, &opts); err != nil {
		return err
	}

	for _, buildRun := range buildRunList.Items {
		if err := r.client.Delete(context.TODO(), &buildRun); err != nil {
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

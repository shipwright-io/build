package build

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	build "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
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
	if err != nil {
		return reconcile.Result{}, err
	}

	// Validate if the spec.output.secretref exist in the namespace
	if b.Spec.Output.SecretRef != nil && b.Spec.Output.SecretRef.Name != "" {
		if err := r.validateOutputSecret(b.Spec.Output.SecretRef.Name, b.Namespace); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Validate if the build strategy is defined
	if b.Spec.StrategyRef != nil {
		if err := r.validateStrategyRef(b.Spec.StrategyRef, b.Namespace); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileBuild) validateStrategyRef(s *build.StrategyRef, ns string) error {
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

package clusterbuildstrategy

import (
	"context"
	"time"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/redhat-developer/build/pkg/ctxlog"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new ClusterBuildStrategy Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(ctx context.Context, mgr manager.Manager) error {
	ctx = ctxlog.NewContext(ctx, "clusterbuildstrategy-controller")
	return add(ctx, mgr, NewReconciler(ctx, mgr))
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(ctx context.Context, mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileClusterBuildStrategy{
		ctx:    ctx,
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(ctx context.Context, mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterbuildstrategy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ClusterBuildStrategy
	err = c.Watch(&source.Kind{Type: &buildv1alpha1.ClusterBuildStrategy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileClusterBuildStrategy implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileClusterBuildStrategy{}

// ReconcileClusterBuildStrategy reconciles a ClusterBuildStrategy object
type ReconcileClusterBuildStrategy struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	ctx    context.Context
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a ClusterBuildStrategy object and makes changes based on the state read
// and what is in the BuildStrategy.Spec
func (r *ReconcileClusterBuildStrategy) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(r.ctx, 300*time.Second)
	defer cancel()

	ctxlog.Info(ctx, "Reconciling ClusterBuildStrategy", "Request.Namespace", request.Namespace, "Request.Name", request.Name)
	return reconcile.Result{}, nil
}

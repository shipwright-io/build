package buildstrategy

import (
	"context"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/redhat-developer/build/pkg/config"
	"github.com/redhat-developer/build/pkg/ctxlog"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new BuildStrategy Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(ctx context.Context, c *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContext(ctx, "buildstrategy-controller")
	return add(ctx, mgr, NewReconciler(ctx, c, mgr))
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(ctx context.Context, c *config.Config, mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileBuildStrategy{
		ctx:    ctx,
		config: c,
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(ctx context.Context, mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("buildstrategy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource BuildStrategy
	err = c.Watch(&source.Kind{Type: &buildv1alpha1.BuildStrategy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileBuildStrategy implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileBuildStrategy{}

// ReconcileBuildStrategy reconciles a BuildStrategy object
type ReconcileBuildStrategy struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	ctx    context.Context
	config *config.Config
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a BuildStrategy object and makes changes based on the state read
// and what is in the BuildStrategy.Spec
func (r *ReconcileBuildStrategy) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(r.ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Info(ctx, "Reconciling BuildStrategy", "Request.Namespace", request.Namespace, "Request.Name", request.Name)
	return reconcile.Result{}, nil
}

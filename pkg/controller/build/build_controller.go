package build

import (
	"context"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	taskv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_build")

const (
	// StrategyBuildpacksv3 is a reference to the name of the strategy use  for buildpacks-v3 builds
	StrategyBuildpacksv3 = "buildpacks-v3"

	// StrategySourceToImage is a reference to the name of the strategy use  for s2i builds
	StrategySourceToImage = "s2i"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Build Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
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

	// TODO: Watch TaskRuns

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
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileBuild) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Build")

	// Fetch the Build instance
	instance := &buildv1alpha1.Build{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
	}

	/*

		TODO: Read "how to build" from a BuildStrategy CR

		s2ibuildStrategy := &buildv1alpha1.BuildStrategy{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: "example-buildstrategy", Namespace: instance.Namespace}, s2ibuildStrategy)
		if err != nil {
			if errors.IsNotFound(err) {
				reqLogger.Info("NOT fetched strategy", "Namespace", s2ibuildStrategy.Namespace, "Name", s2ibuildStrategy.Name)

				// Request object not found, could have been deleted after reconcile request.
				// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
				// Return and don't requeue
				return reconcile.Result{}, nil
			}
		}
		reqLogger.Info("fetched strategy", "Namespace", s2ibuildStrategy.Namespace, "Name", s2ibuildStrategy.Name)
	*/

	var generatedTask *taskv1.Task
	var generatedTaskRun *taskv1.TaskRun

	if instance.Spec.StrategyRef == StrategySourceToImage {
		generatedTask = gets2iTask(instance)
		generatedTaskRun = gets2iTaskRun(instance)
	} else if instance.Spec.StrategyRef == StrategyBuildpacksv3 {
		generatedTask = getBuildpacksV3Task(instance)
		generatedTaskRun = getBuildpacksV3TaskRun(instance)
	}

	if generatedTask == nil && generatedTaskRun == nil {
		return reconcile.Result{}, nil
	}

	if err := controllerutil.SetControllerReference(instance, generatedTask, r.scheme); err != nil {
		log.Error(err, "Setting owner reference fails")
		return reconcile.Result{}, err
	}

	err = r.client.Create(context.TODO(), generatedTask)
	if err != nil {
		reqLogger.Info("failed to create Task", "Namespace", generatedTask.Namespace, "Name", generatedTask.Name)
		return reconcile.Result{}, err
	}

	generatedTaskRun = gets2iTaskRun(instance)

	if err := controllerutil.SetControllerReference(instance, generatedTaskRun, r.scheme); err != nil {
		log.Error(err, "Setting owner reference fails")
		return reconcile.Result{}, err
	}

	err = r.client.Create(context.TODO(), generatedTaskRun)
	if err != nil {
		reqLogger.Info("failed to create TaskRun", "Namespace", generatedTaskRun.Namespace, "Name", generatedTaskRun.Name)

		return reconcile.Result{}, err
	}

	instance.Status = buildv1alpha1.BuildStatus{
		Status: "in-progress",
	}

	err = r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		return reconcile.Result{}, err
	}
	reqLogger.Info("updated Build", "Pod.Namespace", instance.Namespace, "Build.Name", instance.Name)
	return reconcile.Result{}, nil
}

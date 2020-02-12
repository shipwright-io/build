package build

import (
	"context"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	taskv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

	// Watch TaskRuns
	err = c.Watch(&source.Kind{Type: &taskv1.TaskRun{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &buildv1alpha1.Build{},
	})

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

	// Fetch the Build instance
	instance := &buildv1alpha1.Build{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
	}

	var generatedTask *taskv1.Task
	var generatedTaskRun *taskv1.TaskRun

	generatedTaskRun = r.retrieveTaskRun(instance)

	if generatedTaskRun != nil {

		// TODO: Make this safer
		if len(generatedTaskRun.Status.Conditions) > 0 {
			jobStatus := generatedTaskRun.Status.Conditions[0].Reason
			instance.Status.Status = jobStatus
			err = r.client.Status().Update(context.TODO(), instance)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	buildStrategyInstance := r.retrieveCustomBuildStrategy(instance, request)
	if buildStrategyInstance != nil {
		generatedTask = getCustomTask(instance, buildStrategyInstance)
		generatedTaskRun = getCustomTaskRun(instance, buildStrategyInstance)
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
	reqLogger.Info("updated Build", "Build.Namespace", instance.Namespace, "Build.Name", instance.Name)
	return reconcile.Result{}, nil
}

func (r *ReconcileBuild) retrieveCustomBuildStrategy(instance *buildv1alpha1.Build, request reconcile.Request) *buildv1alpha1.BuildStrategy {
	buildStrategyInstance := &buildv1alpha1.BuildStrategy{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.StrategyRef, Namespace: instance.Namespace}, buildStrategyInstance)
	if err != nil {
		log.Error(err, "failed to get BuildStrategy")
		return nil
	}
	return buildStrategyInstance
}

func (r *ReconcileBuild) retrieveTaskRun(instance *buildv1alpha1.Build) *taskv1.TaskRun {

	taskRunList := &taskv1.TaskRunList{}

	lbls := map[string]string{
		"build.dev/build": instance.Name,
	}

	opts := client.ListOptions{
		Namespace:     instance.Namespace,
		LabelSelector: labels.SelectorFromSet(lbls),
	}
	err := r.client.List(context.TODO(), taskRunList, &opts)

	if err != nil {
		log.Error(err, "failed to list existing TaskRuns")
		return nil
	}

	for _, taskRun := range taskRunList.Items {
		return &taskRun
	}
	return nil
}

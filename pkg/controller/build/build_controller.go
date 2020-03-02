package build

import (
	"context"
	"strings"

	taskv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
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

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
)

var log = logf.Log.WithName("controller_build")

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
	return c.Watch(&source.Kind{Type: &taskv1.TaskRun{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &buildv1alpha1.Build{},
	})
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

	// If the reconcile was triggered because of a change in TaskRun,
	// read the TaskRun's status and update Build's status
	existingTaskRun := r.retrieveTaskRun(instance)
	if existingTaskRun != nil {
		// TODO: Make this safer
		if len(existingTaskRun.Status.Conditions) > 0 {
			jobStatus := existingTaskRun.Status.Conditions[0].Reason
			instance.Status.Status = jobStatus
			err = r.client.Status().Update(context.TODO(), instance)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	// Everytime control enters the reconcile loop, we need to ensure
	// everything is in its desired state.
	buildStrategyInstance := r.retrieveCustomBuildStrategy(instance, request)
	if buildStrategyInstance != nil {
		generatedTask = getCustomTask(instance, buildStrategyInstance)
		generatedTaskRun = getCustomTaskRun(instance, buildStrategyInstance)
	}

	if generatedTask == nil && generatedTaskRun == nil {
		return reconcile.Result{}, nil
	}

	existingTask := r.retrieveTask(instance)

	if existingTask != nil && !compare(*existingTask, *generatedTask) {
		// If the Build spec has changed, we must start afresh
		// If the locally generated task's "generation" annotation
		// is different than that of existing task's "generation" annotation,
		// then the Build must have been modified

		err = r.client.Delete(context.TODO(), existingTask)
		if err != nil {
			return reconcile.Result{}, nil
		}

		err = r.client.Delete(context.TODO(), existingTaskRun)
		if err != nil {
			return reconcile.Result{}, nil
		}

		// We've deleted the existing 'jobs', that is,
		// the Task & the TaskRun because they can be considered
		// stale.
	}

	// create Task if no task for that Build exists
	if err := controllerutil.SetControllerReference(instance, generatedTask, r.scheme); err != nil {
		log.Error(err, "Setting owner reference fails")
		return reconcile.Result{}, err
	}

	if r.retrieveTask(instance) == nil {
		err = r.client.Create(context.TODO(), generatedTask)
		if err != nil {
			reqLogger.Info("failed to create Task", "Namespace", generatedTask.Namespace, "Name", generatedTask.Name)
			return reconcile.Result{}, err
		}
	}

	// create Task if no task for that Build exists
	if err := controllerutil.SetControllerReference(instance, generatedTaskRun, r.scheme); err != nil {
		log.Error(err, "Setting owner reference fails")
		return reconcile.Result{}, err
	}

	if r.retrieveTaskRun(instance) == nil {
		// Add creds to service account
		buildServiceAccount, err := r.retrieveServiceAccount(instance, pipelineServiceAccountName)
		if err != nil {
			return reconcile.Result{}, err
		}
		buildServiceAccount = applyCredentials(instance, buildServiceAccount)
		err = r.client.Update(context.TODO(), buildServiceAccount)
		if err != nil {
			log.Error(err, "updating of service account fails")
			return reconcile.Result{}, err
		}

		err = r.client.Create(context.TODO(), generatedTaskRun)
		if err != nil {
			reqLogger.Info("failed to create TaskRun", "Namespace", generatedTaskRun.Namespace, "Name", generatedTaskRun.Name)

			return reconcile.Result{}, err
		}
	}

	reqLogger.Info("Reconciled Build", "Build.Namespace", instance.Namespace, "Build.Name", instance.Name)
	return reconcile.Result{}, nil
}

func compare(a taskv1.Task, b taskv1.Task) bool {
	if a.GetLabels()[labelBuildGeneration] == b.GetLabels()[labelBuildGeneration] {
		return true
	}
	return false
}

func (r *ReconcileBuild) retrieveServiceAccount(instance *buildv1alpha1.Build, pipelineServiceAccountName string) (*corev1.ServiceAccount, error) {
	buildServiceAccount := &corev1.ServiceAccount{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: pipelineServiceAccountName, Namespace: instance.Namespace}, buildServiceAccount)
	if err != nil {
		log.Error(err, "failed to get Service Account")
		return nil, err
	}
	return buildServiceAccount, nil
}

func (r *ReconcileBuild) retrieveCustomBuildStrategy(instance *buildv1alpha1.Build, request reconcile.Request) *buildv1alpha1.BuildStrategy {
	buildStrategyInstance := &buildv1alpha1.BuildStrategy{}
	buildStrategyNameSpace := getBuildStrategyNamespace(instance)

	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.StrategyRef.Name, Namespace: buildStrategyNameSpace}, buildStrategyInstance)
	if err != nil {
		log.Error(err, "failed to get BuildStrategy")
		return nil
	}
	return buildStrategyInstance
}

func getBuildStrategyNamespace(instance *buildv1alpha1.Build) string {
	buildStrategyNameSpace := instance.Namespace
	strategyRefNamespace := strings.TrimSpace(instance.Spec.StrategyRef.Namespace)

	// if namespace is specified in the strategyRef, use it.
	if len(strategyRefNamespace) != 0 {
		buildStrategyNameSpace = strategyRefNamespace
	}
	return buildStrategyNameSpace
}

func (r *ReconcileBuild) retrieveTaskRun(instance *buildv1alpha1.Build) *taskv1.TaskRun {

	taskRunList := &taskv1.TaskRunList{}

	lbls := map[string]string{
		labelBuild: instance.Name,
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

func (r *ReconcileBuild) retrieveTask(instance *buildv1alpha1.Build) *taskv1.Task {

	taskList := &taskv1.TaskList{}

	lbls := map[string]string{
		labelBuild: instance.Name,
	}

	opts := client.ListOptions{
		Namespace:     instance.Namespace,
		LabelSelector: labels.SelectorFromSet(lbls),
	}
	err := r.client.List(context.TODO(), taskList, &opts)

	if err != nil {
		log.Error(err, "failed to list existing TaskRuns")
		return nil
	}

	for _, taskRun := range taskList.Items {
		return &taskRun
	}
	return nil
}

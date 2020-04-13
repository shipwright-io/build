package buildrun

import (
	"context"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	taskv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
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

var log = logf.Log.WithName("controller_buildrun")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new BuildRun Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileBuildRun{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("buildrun-controller", mgr, controller.Options{Reconciler: r})
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

	// Watch for changes to primary resource BuildRun
	err = c.Watch(&source.Kind{Type: &buildv1alpha1.BuildRun{}}, &handler.EnqueueRequestForObject{}, pred)
	if err != nil {
		return err
	}

	// Watch TaskRuns
	return c.Watch(&source.Kind{Type: &taskv1.TaskRun{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &buildv1alpha1.BuildRun{},
	})
}

// blank assignment to verify that ReconcileBuildRun implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileBuildRun{}

// ReconcileBuildRun reconciles a BuildRun object
type ReconcileBuildRun struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Build object and makes changes based on the state read
// and what is in the Build.Spec
func (r *ReconcileBuildRun) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling BuildRun")

	// Fetch the BuildRun instance
	buildRun := &buildv1alpha1.BuildRun{}
	err := r.client.Get(context.TODO(), request.NamespacedName, buildRun)
	if err != nil && !errors.IsNotFound(err) {
		reqLogger.Error(err, "Failed to get the build instance")
		return reconcile.Result{}, err
	} else if errors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}

	// Fetch the Build instance
	build := &buildv1alpha1.Build{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: buildRun.Spec.BuildRef.Name, Namespace: buildRun.Namespace}, build)
	if err != nil {
		reqLogger.Error(err, "Failed to get Build from BuildRun", "Build", buildRun.Spec.BuildRef.Name)
		return reconcile.Result{}, err
	}

	lastTaskRun, err := r.retrieveTaskRun(build, buildRun)
	if err != nil {
		reqLogger.Error(err, "Failed to list existing TaskRuns from BuildRun", "BuildRun", buildRun.Name)
		return reconcile.Result{}, err
	}

	if lastTaskRun != nil {
		// TODO: Make this safer
		if len(lastTaskRun.Status.Conditions) > 0 {
			buildRun.Status.Succeeded = lastTaskRun.Status.Conditions[0].Status
			buildRun.Status.Reason = lastTaskRun.Status.Conditions[0].Reason
		}
		buildRun.Status.LatestTaskRunRef = &lastTaskRun.Name
		buildRun.Status.StartTime = lastTaskRun.Status.StartTime
		buildRun.Status.CompletionTime = lastTaskRun.Status.CompletionTime
		err = r.client.Status().Update(context.TODO(), buildRun)
		if err != nil {
			reqLogger.Error(err, "Failed to update the BuildRun status", "BuildRun", buildRun.Name)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// Add creds to service account
	serviceAccount, err := r.retrieveServiceAccount(buildRun)
	if err != nil {
		reqLogger.Error(err, "Failed to get ServiceAccount from BuildRun", "BuildRun", buildRun.Name)
		return reconcile.Result{}, err
	}
	serviceAccount = applyCredentials(build, serviceAccount)
	err = r.client.Update(context.TODO(), serviceAccount)
	if err != nil {
		reqLogger.Error(err, "Failed to update ServiceAccount", "ServiceAccount", serviceAccount.Name)
		return reconcile.Result{}, err
	}

	// Everytime control enters the reconcile loop, we need to ensure
	// everything is in its desired state.
	var generatedTaskRun *taskv1.TaskRun
	if build.Spec.StrategyRef.Kind == nil || *build.Spec.StrategyRef.Kind == buildv1alpha1.NamespacedBuildStrategyKind {
		buildStrategy := r.retrieveBuildStrategy(build, request)
		if buildStrategy != nil {
			generatedTaskRun, err = generateTaskRun(build, buildRun, serviceAccount.Name, buildStrategy.Spec.BuildSteps)
			if err != nil {
				reqLogger.Error(err, "Failed to generate TaskRun", "BuildRun", buildRun.Name)
				return reconcile.Result{}, err
			}
		}
	} else if *build.Spec.StrategyRef.Kind == buildv1alpha1.ClusterBuildStrategyKind {
		clusterBuildStrategy := r.retrieveClusterBuildStrategy(build, request)
		if clusterBuildStrategy != nil {
			generatedTaskRun, err = generateTaskRun(build, buildRun, serviceAccount.Name, clusterBuildStrategy.Spec.BuildSteps)
			if err != nil {
				reqLogger.Error(err, "Failed to generate TaskRun", "BuildRun", buildRun.Name)
				return reconcile.Result{}, err
			}
		}
	} else {
		log.Error(err, "Unsupported BuildStrategy Kind", "BuildStrategyKind", build.Spec.StrategyRef.Kind)
		return reconcile.Result{}, err
	}

	// Set OwnerReference for Build and BuildRun
	if err := controllerutil.SetControllerReference(build, buildRun, r.scheme); err != nil {
		reqLogger.Error(err, "Setting owner reference fails", "Build", build.Name, "BuildRun", buildRun.Name)
		return reconcile.Result{}, err
	}

	// Set OwnerReference for BuildRun and TaskRun
	if err := controllerutil.SetControllerReference(buildRun, generatedTaskRun, r.scheme); err != nil {
		reqLogger.Error(err, "Setting owner reference fails", "BuildRun", buildRun.Name, "TaskRun", generatedTaskRun.Name)
		return reconcile.Result{}, err
	}

	// create TaskRun if no TaskRun for that BuildRun exists
	err = r.client.Create(context.TODO(), generatedTaskRun)
	if err != nil {
		reqLogger.Error(err, "Failed to create TaskRun", "Namespace", generatedTaskRun.Namespace, "Name", generatedTaskRun.Name)
		return reconcile.Result{}, err
	}

	reqLogger.Info("Generate and create TaskRun from Build and BuildRun", "TaskRun", generatedTaskRun.Name, "Build", build.Name, "BuildRun", buildRun.Name)
	reqLogger.Info("Reconciled Build", "Build.Namespace", buildRun.Namespace, "Build.Name", buildRun.Name)
	return reconcile.Result{}, nil
}

// IsRunning return if the TaskRun is running
func isTaskRunRunning(tr *taskv1.TaskRun) bool {
	if tr == nil {
		return false
	}
	return tr.Status.GetCondition(apis.ConditionSucceeded).IsUnknown()
}

func (r *ReconcileBuildRun) retrieveServiceAccount(buildRun *buildv1alpha1.BuildRun) (*corev1.ServiceAccount, error) {
	buildServiceAccount := &corev1.ServiceAccount{}

	// If ServiceAccount in Build Spec is nil, use default serviceaccount
	var serviceAccountName string
	if buildRun.Spec.ServiceAccount == nil {
		serviceAccountName = pipelineServiceAccountName
	} else {
		serviceAccountName = *buildRun.Spec.ServiceAccount
	}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: serviceAccountName, Namespace: buildRun.Namespace}, buildServiceAccount)
	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Failed to get ServiceAccount", "ServiceAccount", serviceAccountName)
		return nil, err
	} else if errors.IsNotFound(err) {
		serviceAccountName = defaultServiceAccountName
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: serviceAccountName, Namespace: buildRun.Namespace}, buildServiceAccount)
		if err != nil {
			log.Error(err, "Failed to get default ServiceAccount")
			return nil, err
		}
	}
	log.Info("Retrieve ServiceAccount from BuildRun", "ServiceAccount", serviceAccountName)
	return buildServiceAccount, nil
}

func (r *ReconcileBuildRun) retrieveBuildStrategy(instance *buildv1alpha1.Build, request reconcile.Request) *buildv1alpha1.BuildStrategy {
	buildStrategyInstance := &buildv1alpha1.BuildStrategy{}

	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.StrategyRef.Name, Namespace: instance.Namespace}, buildStrategyInstance)
	if err != nil {
		log.Error(err, "Failed to get BuildStrategy")
		return nil
	}
	return buildStrategyInstance
}

func (r *ReconcileBuildRun) retrieveClusterBuildStrategy(instance *buildv1alpha1.Build, request reconcile.Request) *buildv1alpha1.ClusterBuildStrategy {
	clusterBuildStrategyInstance := &buildv1alpha1.ClusterBuildStrategy{}

	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.StrategyRef.Name}, clusterBuildStrategyInstance)
	if err != nil {
		log.Error(err, "Failed to get ClusterBuildStrategy")
		return nil
	}
	return clusterBuildStrategyInstance
}

func (r *ReconcileBuildRun) retrieveTaskRun(build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun) (*taskv1.TaskRun, error) {

	taskRunList := &taskv1.TaskRunList{}

	lbls := map[string]string{
		buildv1alpha1.LabelBuild:    build.Name,
		buildv1alpha1.LabelBuildRun: buildRun.Name,
	}
	opts := client.ListOptions{
		Namespace:     buildRun.Namespace,
		LabelSelector: labels.SelectorFromSet(lbls),
	}
	err := r.client.List(context.TODO(), taskRunList, &opts)

	if err != nil {
		return nil, err
	}

	if len(taskRunList.Items) > 0 {
		return &taskRunList.Items[len(taskRunList.Items)-1], nil
	}
	return nil, nil
}

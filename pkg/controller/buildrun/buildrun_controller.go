package buildrun

import (
	"context"
	"fmt"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type setOwnerReferenceFunc func(owner, object metav1.Object, scheme *runtime.Scheme) error

// Add creates a new BuildRun Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, NewReconciler(mgr, controllerutil.SetControllerReference))
}

// blank assignment to verify that ReconcileBuildRun implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileBuildRun{}

// ReconcileBuildRun reconciles a BuildRun object
type ReconcileBuildRun struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client                client.Client
	scheme                *runtime.Scheme
	setOwnerReferenceFunc setOwnerReferenceFunc
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(mgr manager.Manager, ownerRef setOwnerReferenceFunc) reconcile.Reconciler {
	return &ReconcileBuildRun{
		client:                mgr.GetClient(),
		scheme:                mgr.GetScheme(),
		setOwnerReferenceFunc: ownerRef,
	}
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
	return c.Watch(&source.Kind{Type: &v1beta1.TaskRun{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &buildv1alpha1.BuildRun{},
	})
}

// Reconcile reads that state of the cluster for a Build object and makes changes based on the state read
// and what is in the Build.Spec
func (r *ReconcileBuildRun) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling BuildRun")

	// Fetch the BuildRun instance
	buildRun := &buildv1alpha1.BuildRun{}
	err := r.client.Get(context.TODO(), request.NamespacedName, buildRun)
	if err != nil && !apierrors.IsNotFound(err) {
		return reconcile.Result{}, err
	} else if apierrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}

	// Fetch the Build instance
	build := &buildv1alpha1.Build{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: buildRun.Spec.BuildRef.Name, Namespace: buildRun.Namespace}, build)
	if err != nil {
		updateErr := r.updateBuildRunErrorStatus(buildRun, err.Error())
		return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
	}

	lastTaskRun, err := r.retrieveTaskRun(build, buildRun)
	if err != nil {
		reqLogger.Error(err, "Failed to list existing TaskRuns from BuildRun", "BuildRun", buildRun.Name)
		updateErr := r.updateBuildRunErrorStatus(buildRun, err.Error())
		return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
	}

	if lastTaskRun != nil {
		// TODO: Make this safer
		if len(lastTaskRun.Status.Conditions) > 0 {
			taskRunStatus := lastTaskRun.Status.Conditions[0].Status
			buildRun.Status.Succeeded = taskRunStatus
			if taskRunStatus == corev1.ConditionFalse {
				buildRun.Status.Reason = lastTaskRun.Status.Conditions[0].Message
			} else {
				buildRun.Status.Reason = lastTaskRun.Status.Conditions[0].Reason
			}
		}
		buildRun.Status.LatestTaskRunRef = &lastTaskRun.Name
		buildRun.Status.StartTime = lastTaskRun.Status.StartTime
		buildRun.Status.CompletionTime = lastTaskRun.Status.CompletionTime
		if buildRun.Status.BuildSpec == nil {
			buildRun.Status.BuildSpec = &build.Spec
		}
		err = r.client.Status().Update(context.TODO(), buildRun)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// Choose a service account to use
	serviceAccount, err := r.retrieveServiceAccount(build, buildRun)
	if err != nil {
		updateErr := r.updateBuildRunErrorStatus(buildRun, err.Error())
		return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
	}

	// Everytime control enters the reconcile loop, we need to ensure
	// everything is in its desired state.
	var generatedTaskRun *v1beta1.TaskRun
	if build.Spec.StrategyRef.Kind == nil || *build.Spec.StrategyRef.Kind == buildv1alpha1.NamespacedBuildStrategyKind {
		buildStrategy, err := r.retrieveBuildStrategy(build, request)
		if err != nil {
			return reconcile.Result{}, err
		}
		if buildStrategy != nil {
			generatedTaskRun, err = generateTaskRun(build, buildRun, serviceAccount.Name, buildStrategy.Spec.BuildSteps)
			if err != nil {
				updateErr := r.updateBuildRunErrorStatus(buildRun, err.Error())
				return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
			}
		}
	} else if *build.Spec.StrategyRef.Kind == buildv1alpha1.ClusterBuildStrategyKind {
		clusterBuildStrategy, err := r.retrieveClusterBuildStrategy(build, request)
		if err != nil {
			return reconcile.Result{}, err
		}
		if clusterBuildStrategy != nil {
			generatedTaskRun, err = generateTaskRun(build, buildRun, serviceAccount.Name, clusterBuildStrategy.Spec.BuildSteps)
			if err != nil {
				updateErr := r.updateBuildRunErrorStatus(buildRun, err.Error())
				return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
			}
		}
	} else {
		err := fmt.Errorf("unknown strategy %s", string(*build.Spec.StrategyRef.Kind))
		reqLogger.Error(err, "Unsupported BuildStrategy Kind", "BuildStrategyKind", build.Spec.StrategyRef.Kind)
		updateErr := r.updateBuildRunErrorStatus(buildRun, err.Error())
		return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
	}

	// Set OwnerReference for Build and BuildRun
	if err := r.setOwnerReferenceFunc(build, buildRun, r.scheme); err != nil {
		updateErr := r.updateBuildRunErrorStatus(buildRun, err.Error())
		return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
	}

	// Set OwnerReference for BuildRun and TaskRun
	if err := r.setOwnerReferenceFunc(buildRun, generatedTaskRun, r.scheme); err != nil {
		updateErr := r.updateBuildRunErrorStatus(buildRun, err.Error())
		return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
	}

	// create TaskRun if no TaskRun for that BuildRun exists
	err = r.client.Create(context.TODO(), generatedTaskRun)
	if err != nil {
		updateErr := r.updateBuildRunErrorStatus(buildRun, err.Error())
		return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
	}

	reqLogger.Info("Generate and create TaskRun from Build and BuildRun", "TaskRun", generatedTaskRun.Name, "Build", build.Name, "BuildRun", buildRun.Name)
	reqLogger.Info("Reconciled Build", "Build.Namespace", buildRun.Namespace, "Build.Name", buildRun.Name)
	return reconcile.Result{}, nil
}

// IsRunning return if the TaskRun is running
func isTaskRunRunning(tr *v1beta1.TaskRun) bool {
	if tr == nil {
		return false
	}
	return tr.Status.GetCondition(apis.ConditionSucceeded).IsUnknown()
}

func (r *ReconcileBuildRun) retrieveServiceAccount(build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun) (*corev1.ServiceAccount, error) {
	serviceAccount := &corev1.ServiceAccount{}
	serviceAccountName := buildRun.Name + "-sa"
	if buildRun.Spec.ServiceAccount != nil && buildRun.Spec.ServiceAccount.Generate == true {
		serviceAccount.Name = serviceAccountName
		serviceAccount.Namespace = buildRun.Namespace
		serviceAccount.Labels = map[string]string{buildv1alpha1.LabelBuildRun: buildRun.Name}
		ownerReferences := metav1.NewControllerRef(buildRun, buildv1alpha1.SchemeGroupVersion.WithKind("BuildRun"))
		serviceAccount.OwnerReferences = append(serviceAccount.OwnerReferences, *ownerReferences)

		// Add credentials and create the new service account
		serviceAccount = ApplyCredentials(build, serviceAccount)
		err := r.client.Create(context.TODO(), serviceAccount)
		if err != nil {
			return nil, err
		}
		log.Info("Generate a new ServiceAccount for BuildRun", "ServiceAccount", serviceAccount.Name)
	} else {
		// If ServiceAccount or the name of ServiceAccount in buildRun is nil, use pipeline serviceaccount
		if buildRun.Spec.ServiceAccount == nil || buildRun.Spec.ServiceAccount.Name == nil {
			serviceAccountName = pipelineServiceAccountName
			err := r.client.Get(context.TODO(), types.NamespacedName{Name: serviceAccountName, Namespace: buildRun.Namespace}, serviceAccount)
			if err != nil && !apierrors.IsNotFound(err) {
				return nil, err
			} else if apierrors.IsNotFound(err) {
				serviceAccountName = defaultServiceAccountName
				err = r.client.Get(context.TODO(), types.NamespacedName{Name: serviceAccountName, Namespace: buildRun.Namespace}, serviceAccount)
				if err != nil {
					return nil, err
				}
			}
		} else {
			serviceAccountName = *buildRun.Spec.ServiceAccount.Name
			err := r.client.Get(context.TODO(), types.NamespacedName{Name: serviceAccountName, Namespace: buildRun.Namespace}, serviceAccount)
			if err != nil {
				return nil, err
			}
		}

		// Add credentials and update the service account
		serviceAccount = ApplyCredentials(build, serviceAccount)
		err := r.client.Update(context.TODO(), serviceAccount)
		if err != nil {
			return nil, err
		}
		log.Info("Retrieve ServiceAccount from BuildRun", "ServiceAccount", serviceAccount.Name)
	}
	return serviceAccount, nil
}

func (r *ReconcileBuildRun) retrieveBuildStrategy(instance *buildv1alpha1.Build, request reconcile.Request) (*buildv1alpha1.BuildStrategy, error) {
	buildStrategyInstance := &buildv1alpha1.BuildStrategy{}

	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.StrategyRef.Name, Namespace: instance.Namespace}, buildStrategyInstance)
	if err != nil {
		log.Error(err, "Failed to get BuildStrategy")
		return nil, err
	}
	return buildStrategyInstance, nil
}

func (r *ReconcileBuildRun) retrieveClusterBuildStrategy(instance *buildv1alpha1.Build, request reconcile.Request) (*buildv1alpha1.ClusterBuildStrategy, error) {
	clusterBuildStrategyInstance := &buildv1alpha1.ClusterBuildStrategy{}

	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.StrategyRef.Name}, clusterBuildStrategyInstance)
	if err != nil {
		log.Error(err, "Failed to get ClusterBuildStrategy")
		return nil, err
	}
	return clusterBuildStrategyInstance, nil
}

func (r *ReconcileBuildRun) retrieveTaskRun(build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun) (*v1beta1.TaskRun, error) {

	taskRunList := &v1beta1.TaskRunList{}

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

func (r *ReconcileBuildRun) updateBuildRunErrorStatus(buildRun *buildv1alpha1.BuildRun, errorMessage string) error {
	buildRun.Status.Succeeded = corev1.ConditionFalse
	buildRun.Status.Reason = errorMessage
	now := metav1.Now()
	buildRun.Status.StartTime = &now
	updateErr := r.client.Status().Update(context.TODO(), buildRun)
	return updateErr
}

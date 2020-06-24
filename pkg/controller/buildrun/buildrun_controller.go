package buildrun

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/redhat-developer/build/pkg/config"
	"github.com/redhat-developer/build/pkg/ctxlog"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
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
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const namespace string = "namespace"
const name string = "name"

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

type setOwnerReferenceFunc func(owner, object metav1.Object, scheme *runtime.Scheme) error

// Add creates a new BuildRun Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(ctx context.Context, c *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContext(ctx, "buildrun-controller")
	return add(ctx, mgr, NewReconciler(ctx, c, mgr, controllerutil.SetControllerReference))
}

// blank assignment to verify that ReconcileBuildRun implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileBuildRun{}

// ReconcileBuildRun reconciles a BuildRun object
type ReconcileBuildRun struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	ctx                   context.Context
	config                *config.Config
	client                client.Client
	scheme                *runtime.Scheme
	setOwnerReferenceFunc setOwnerReferenceFunc
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(ctx context.Context, c *config.Config, mgr manager.Manager, ownerRef setOwnerReferenceFunc) reconcile.Reconciler {
	return &ReconcileBuildRun{
		ctx:                   ctx,
		config:                c,
		client:                mgr.GetClient(),
		scheme:                mgr.GetScheme(),
		setOwnerReferenceFunc: ownerRef,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(ctx context.Context, mgr manager.Manager, r reconcile.Reconciler) error {
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

// This function only returns multiple errors if each error is not nil.
// And its error message.
func handleError(message string, listOfErrors ...error) error {
	var errSlice []string
	for _, e := range listOfErrors {
		if e != nil {
			errSlice = append(errSlice, e.Error())
		}
	}
	return fmt.Errorf("errors: %s, msg: %s", strings.Join(errSlice, ", "), message)
}

// Reconcile reads that state of the cluster for a Build object and makes changes based on the state read
// and what is in the Build.Spec
func (r *ReconcileBuildRun) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(r.ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Debug(ctx, "start reconciling BuildRun", namespace, request.Namespace, name, request.Name)

	// Fetch the BuildRun instance
	buildRun := &buildv1alpha1.BuildRun{}
	err := r.client.Get(ctx, request.NamespacedName, buildRun)
	if err != nil && !apierrors.IsNotFound(err) {
		return reconcile.Result{}, err
	} else if apierrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}

	// Fetch the Build instance
	build := &buildv1alpha1.Build{}
	err = r.client.Get(ctx, types.NamespacedName{Name: buildRun.Spec.BuildRef.Name, Namespace: buildRun.Namespace}, build)
	if err != nil {
		updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
		return reconcile.Result{}, handleError("Failed to fetch the Build instance", err, updateErr)
	}

	if buildRun.GetLabels() == nil {
		buildRun.Labels = make(map[string]string)
	}
	if buildRun.GetLabels()[buildv1alpha1.LabelBuild] == "" {
		buildRun.Labels[buildv1alpha1.LabelBuild] = build.Name
		buildRun.Labels[buildv1alpha1.LabelBuildGeneration] = strconv.FormatInt(build.Generation, 10)
		ctxlog.Info(ctx, "updating buildRun labels", namespace, request.Namespace, name, request.Name)
		err = r.client.Update(ctx, buildRun)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	lastTaskRun, err := r.retrieveTaskRun(ctx, build, buildRun)
	if err != nil {
		updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
		return reconcile.Result{}, handleError(fmt.Sprintf("Failed to list existing TaskRuns from BuildRun: %v", buildRun.Name), err, updateErr)
	}

	if lastTaskRun != nil {
		// TODO: Make this safer
		if len(lastTaskRun.Status.Conditions) > 0 {
			taskRunStatus := lastTaskRun.Status.Conditions[0].Status

			// check if we should delete the generated service account by checking the build run spec and that the task run is complete
			if isGeneratedServiceAccountUsed(buildRun) && (taskRunStatus == corev1.ConditionTrue || taskRunStatus == corev1.ConditionFalse) {
				serviceAccount := &corev1.ServiceAccount{}
				serviceAccount.Name = getGeneratedServiceAccountName(buildRun)
				serviceAccount.Namespace = buildRun.Namespace

				ctxlog.Info(ctx, "deleting service account", namespace, request.Namespace, name, request.Name)
				if err = r.client.Delete(ctx, serviceAccount); err != nil && !apierrors.IsNotFound(err) {
					ctxlog.Error(ctx, err, "Error during deletion of generated service account.")
					return reconcile.Result{}, err
				}
			}

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

		ctxlog.Info(ctx, "updating buildRun status", namespace, request.Namespace, name, request.Name)
		err = r.client.Status().Update(ctx, buildRun)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// Choose a service account to use
	serviceAccount, err := r.retrieveServiceAccount(ctx, build, buildRun)
	if err != nil {
		updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
		return reconcile.Result{}, handleError("Failed to choose a service account to use", err, updateErr)
	}

	// Everytime control enters the reconcile loop, we need to ensure
	// everything is in its desired state.
	var generatedTaskRun *v1beta1.TaskRun
	if build.Spec.StrategyRef.Kind == nil || *build.Spec.StrategyRef.Kind == buildv1alpha1.NamespacedBuildStrategyKind {
		buildStrategy, err := r.retrieveBuildStrategy(ctx, build, request)
		if err != nil {
			return reconcile.Result{}, err
		}
		if buildStrategy != nil {
			generatedTaskRun, err = GenerateTaskRun(build, buildRun, serviceAccount.Name, buildStrategy.Spec.BuildSteps)
			if err != nil {
				updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
				return reconcile.Result{}, handleError("Failed to generate the taskrun with buildStrategy", err, updateErr)
			}
		}
	} else if *build.Spec.StrategyRef.Kind == buildv1alpha1.ClusterBuildStrategyKind {
		clusterBuildStrategy, err := r.retrieveClusterBuildStrategy(ctx, build, request)
		if err != nil {
			return reconcile.Result{}, err
		}
		if clusterBuildStrategy != nil {
			generatedTaskRun, err = GenerateTaskRun(build, buildRun, serviceAccount.Name, clusterBuildStrategy.Spec.BuildSteps)
			if err != nil {
				updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
				return reconcile.Result{}, handleError("Failed to generate the taskrun with clusterBuildStrategy", err, updateErr)
			}
		}
	} else {
		err := fmt.Errorf("unknown strategy %s", string(*build.Spec.StrategyRef.Kind))
		updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
		return reconcile.Result{}, handleError(fmt.Sprintf("Unsupported BuildStrategy Kind: %v", build.Spec.StrategyRef.Kind), err, updateErr)
	}

	// Set OwnerReference for Build and BuildRun
	if err := r.setOwnerReferenceFunc(build, buildRun, r.scheme); err != nil {
		updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
		return reconcile.Result{}, handleError("Failed to set OwnerReference for Build and BuildRun", err, updateErr)
	}

	// Set OwnerReference for BuildRun and TaskRun
	if err := r.setOwnerReferenceFunc(buildRun, generatedTaskRun, r.scheme); err != nil {
		updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
		return reconcile.Result{}, handleError("Failed to set OwnerReference for BuildRun and TaskRun", err, updateErr)
	}

	// create TaskRun if no TaskRun for that BuildRun exists
	ctxlog.Info(ctx, "creating TaskRun from BuildRun", namespace, request.Namespace, name, generatedTaskRun.Name, "BuildRun", buildRun.Name)
	err = r.client.Create(ctx, generatedTaskRun)
	if err != nil {
		updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
		return reconcile.Result{}, handleError("Failed to create TaskRun if no TaskRun for that BuildRun exists", err, updateErr)
	}

	ctxlog.Debug(ctx, "finishing reconciling BuildRun", namespace, request.Namespace, name, request.Name)

	return reconcile.Result{}, nil
}

// IsRunning return if the TaskRun is running
func isTaskRunRunning(tr *v1beta1.TaskRun) bool {
	if tr == nil {
		return false
	}
	return tr.Status.GetCondition(apis.ConditionSucceeded).IsUnknown()
}

func (r *ReconcileBuildRun) retrieveServiceAccount(ctx context.Context, build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun) (*corev1.ServiceAccount, error) {
	serviceAccount := &corev1.ServiceAccount{}

	if isGeneratedServiceAccountUsed(buildRun) {
		serviceAccountName := getGeneratedServiceAccountName(buildRun)

		serviceAccount.Name = serviceAccountName
		serviceAccount.Namespace = buildRun.Namespace

		// Create the service account, use CreateOrUpdate as it might exist already from a previous reconcilation that
		// succeeded to create the service account but failed to update the build run that references it
		ctxlog.Info(ctx, "create or update serviceAccount for BuildRun", namespace, buildRun.Namespace, name, serviceAccountName, "BuildRun", buildRun.Name)
		op, err := controllerutil.CreateOrUpdate(ctx, r.client, serviceAccount, func() error {
			serviceAccount.SetLabels(map[string]string{buildv1alpha1.LabelBuildRun: buildRun.Name})

			ownerReference := metav1.NewControllerRef(buildRun, buildv1alpha1.SchemeGroupVersion.WithKind("BuildRun"))
			serviceAccount.SetOwnerReferences([]metav1.OwnerReference{*ownerReference})

			ApplyCredentials(ctx, build, serviceAccount)

			return nil
		})
		if err != nil {
			return nil, err
		}
		ctxlog.Debug(ctx, "Automatic generation of service account", namespace, serviceAccount.Namespace, name, serviceAccount.Name, "Operation", op)
	} else {
		// If ServiceAccount or the name of ServiceAccount in buildRun is nil, use pipeline serviceaccount
		if buildRun.Spec.ServiceAccount == nil || buildRun.Spec.ServiceAccount.Name == nil {
			serviceAccountName := pipelineServiceAccountName
			err := r.client.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: buildRun.Namespace}, serviceAccount)
			if err != nil && !apierrors.IsNotFound(err) {
				return nil, err
			} else if apierrors.IsNotFound(err) {
				serviceAccountName = defaultServiceAccountName
				ctxlog.Info(ctx, "falling back to default serviceAccount", namespace, buildRun.Namespace)
				err = r.client.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: buildRun.Namespace}, serviceAccount)
				if err != nil {
					return nil, err
				}
			}
		} else {
			serviceAccountName := *buildRun.Spec.ServiceAccount.Name
			err := r.client.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: buildRun.Namespace}, serviceAccount)
			if err != nil {
				return nil, err
			}
		}

		// Add credentials and update the service account
		if modified := ApplyCredentials(ctx, build, serviceAccount); modified {
			ctxlog.Info(ctx, "updating ServiceAccount with secrets from build", namespace, serviceAccount.Namespace, name, serviceAccount.Name)
			if err := r.client.Update(ctx, serviceAccount); err != nil {
				return nil, err
			}
		}
	}
	return serviceAccount, nil
}

func (r *ReconcileBuildRun) retrieveBuildStrategy(ctx context.Context, build *buildv1alpha1.Build, request reconcile.Request) (*buildv1alpha1.BuildStrategy, error) {
	buildStrategyInstance := &buildv1alpha1.BuildStrategy{}

	ctxlog.Debug(ctx, "retrieving BuildStrategy", namespace, build.Namespace, name, build.Name)
	err := r.client.Get(ctx, types.NamespacedName{Name: build.Spec.StrategyRef.Name, Namespace: build.Namespace}, buildStrategyInstance)
	if err != nil {
		return nil, err
	}
	return buildStrategyInstance, nil
}

func (r *ReconcileBuildRun) retrieveClusterBuildStrategy(ctx context.Context, build *buildv1alpha1.Build, request reconcile.Request) (*buildv1alpha1.ClusterBuildStrategy, error) {
	clusterBuildStrategyInstance := &buildv1alpha1.ClusterBuildStrategy{}

	ctxlog.Debug(ctx, "retrieving ClusterBuildStrategy", namespace, build.Namespace, name, build.Name)
	err := r.client.Get(ctx, types.NamespacedName{Name: build.Spec.StrategyRef.Name}, clusterBuildStrategyInstance)
	if err != nil {
		return nil, err
	}
	return clusterBuildStrategyInstance, nil
}

func (r *ReconcileBuildRun) retrieveTaskRun(ctx context.Context, build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun) (*v1beta1.TaskRun, error) {

	taskRunList := &v1beta1.TaskRunList{}

	lbls := map[string]string{
		buildv1alpha1.LabelBuild:    build.Name,
		buildv1alpha1.LabelBuildRun: buildRun.Name,
	}
	opts := client.ListOptions{
		Namespace:     buildRun.Namespace,
		LabelSelector: labels.SelectorFromSet(lbls),
	}

	ctxlog.Info(ctx, "listing taskruns", namespace, buildRun.Namespace, name, buildRun.Name)
	if err := r.client.List(ctx, taskRunList, &opts); err != nil {
		return nil, err
	}

	if len(taskRunList.Items) > 0 {
		return &taskRunList.Items[len(taskRunList.Items)-1], nil
	}
	return nil, nil
}

func (r *ReconcileBuildRun) updateBuildRunErrorStatus(ctx context.Context, buildRun *buildv1alpha1.BuildRun, errorMessage string) error {
	buildRun.Status.Succeeded = corev1.ConditionFalse
	buildRun.Status.Reason = errorMessage
	now := metav1.Now()
	buildRun.Status.StartTime = &now
	ctxlog.Debug(ctx, "updating buildRun status", namespace, buildRun.Namespace, name, buildRun.Name)
	updateErr := r.client.Status().Update(ctx, buildRun)
	return updateErr
}

// getGeneratedServiceAccountName returns the name of the generated service account for a build run
func getGeneratedServiceAccountName(buildRun *buildv1alpha1.BuildRun) string {
	return buildRun.Name + "-sa"
}

// isGeneratedServiceAccountUsed checks if a build run uses a generated service account
func isGeneratedServiceAccountUsed(buildRun *buildv1alpha1.BuildRun) bool {
	return buildRun.Spec.ServiceAccount != nil && buildRun.Spec.ServiceAccount.Generate == true
}

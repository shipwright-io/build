// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/apis"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	buildmetrics "github.com/shipwright-io/build/pkg/metrics"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
)

const (
	namespace          string = "namespace"
	name               string = "name"
	generatedNameRegex        = "-[a-z0-9]{5,5}$"
)

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

// Reconcile reads that state of the cluster for a Build object and makes changes based on the state read
// and what is in the Build.Spec
func (r *ReconcileBuildRun) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	var buildRun *buildv1alpha1.BuildRun
	var build *buildv1alpha1.Build

	updateBuildRunRequired := false

	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(r.ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Debug(ctx, "starting reconciling request from a BuildRun or TaskRun event", namespace, request.Namespace, name, request.Name)

	buildRun = &buildv1alpha1.BuildRun{}
	lastTaskRun := &v1beta1.TaskRun{}

	// for existing TaskRuns update the BuildRun Status, if there is no TaskRun, then create one
	if err := r.client.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, lastTaskRun); err != nil {
		if apierrors.IsNotFound(err) {
			err = r.GetBuildRunObject(ctx, request.Name, request.Namespace, buildRun)
			if err != nil && !apierrors.IsNotFound(err) {
				return reconcile.Result{}, err
			} else if apierrors.IsNotFound(err) {
				// If the BuildRun and TaskRun are not found, it might mean that we are running a Reconcile after a TaskRun was deleted. If this is the case, we need
				// to identify from the request the BuildRun name associate to it and update the BuildRun Status.
				r.VerifyRequestName(ctx, request, buildRun)
				return reconcile.Result{}, nil
			}

			build = &buildv1alpha1.Build{}
			if err = resources.GetBuildObject(ctx, r.client, buildRun.Spec.BuildRef.Name, buildRun.Namespace, build); err != nil {
				updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
				return reconcile.Result{}, resources.HandleError("Failed to fetch the Build instance", err, updateErr)
			}

			// Validate if the Build was successfully registered
			if err := r.ValidateBuildRegistration(ctx, build, buildRun); err != nil {
				return reconcile.Result{}, err
			}

			// Ensure the build-related labels on the BuildRun
			if buildRun.GetLabels() == nil {
				buildRun.Labels = make(map[string]string)
			}

			// Set OwnerReference for Build and BuildRun only when build.build.dev/build-run-deletion is set "true"
			if build.GetAnnotations()[buildv1alpha1.AnnotationBuildRunDeletion] == "true" && !resources.IsOwnedByBuild(build, buildRun.OwnerReferences) {
				if err := r.setOwnerReferenceFunc(build, buildRun, r.scheme); err != nil {
					build.Status.Reason = buildv1alpha1.SetOwnerReferenceFailed
					build.Status.Message = fmt.Sprintf("unexpected error when trying to set the ownerreference: %v", err)
					if err := r.client.Status().Update(ctx, build); err != nil {
						return reconcile.Result{}, err
					}
				}
				ctxlog.Info(ctx, fmt.Sprintf("updating BuildRun %s OwnerReferences, owner is Build %s", buildRun.Name, build.Name), namespace, request.Namespace, name, request.Name)
				updateBuildRunRequired = true
			}

			buildGeneration := strconv.FormatInt(build.Generation, 10)
			if buildRun.GetLabels()[buildv1alpha1.LabelBuild] != build.Name || buildRun.GetLabels()[buildv1alpha1.LabelBuildGeneration] != buildGeneration {
				buildRun.Labels[buildv1alpha1.LabelBuild] = build.Name
				buildRun.Labels[buildv1alpha1.LabelBuildGeneration] = buildGeneration
				ctxlog.Info(ctx, "updating BuildRun labels", namespace, request.Namespace, name, request.Name)
				updateBuildRunRequired = true
			}

			if updateBuildRunRequired {
				if err = r.client.Update(ctx, buildRun); err != nil {
					return reconcile.Result{}, err
				}
				ctxlog.Info(ctx, fmt.Sprintf("successfully updated BuildRun %s", buildRun.Name), namespace, request.Namespace, name, request.Name)
			}

			// Set the Build spec in the BuildRun status
			buildRun.Status.BuildSpec = &build.Spec
			ctxlog.Info(ctx, "updating BuildRun status", namespace, request.Namespace, name, request.Name)
			if err = r.client.Status().Update(ctx, buildRun); err != nil {
				return reconcile.Result{}, err
			}

			// Create the TaskRun, this needs to be the last step in this block to be idempotent
			generatedTaskRun, err := r.createTaskRun(ctx, build, buildRun)
			if err != nil {
				return reconcile.Result{}, err
			}

			ctxlog.Info(ctx, "creating TaskRun from BuildRun", namespace, request.Namespace, name, generatedTaskRun.GenerateName, "BuildRun", buildRun.Name)
			if err = r.client.Create(ctx, generatedTaskRun); err != nil {
				updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
				return reconcile.Result{}, resources.HandleError("Failed to create TaskRun if no TaskRun for that BuildRun exists", err, updateErr)
			}

			// Set the LastTaskRunRef in the BuildRun status
			buildRun.Status.LatestTaskRunRef = &generatedTaskRun.Name
			ctxlog.Info(ctx, "updating BuildRun status with TaskRun name", namespace, request.Namespace, name, request.Name, "TaskRun", generatedTaskRun.Name)
			if err = r.client.Status().Update(ctx, buildRun); err != nil {
				// we ignore the error here to prevent another reconciliation that would create another TaskRun,
				// the LatestTaskRunRef field will also be set in the reconciliation from a TaskRun
				// risk is that when the controller is now restarted before the field is set, another TaskRun will be created
				ctxlog.Error(ctx, err, "Failed to update BuildRun status is ignored", namespace, request.Namespace, name, request.Name)
			}

			// Increase BuildRun count in metrics
			buildmetrics.BuildRunCountInc(buildRun.Status.BuildSpec.StrategyRef.Name, buildRun.Namespace, buildRun.Spec.BuildRef.Name, buildRun.Name)

			// Report buildrun ramp-up duration (time between buildrun creation and taskrun creation)
			buildmetrics.BuildRunRampUpDurationObserve(
				buildRun.Status.BuildSpec.StrategyRef.Name,
				buildRun.Namespace,
				buildRun.Spec.BuildRef.Name,
				buildRun.Name,
				generatedTaskRun.CreationTimestamp.Time.Sub(buildRun.CreationTimestamp.Time),
			)
		} else {
			return reconcile.Result{}, err
		}
	} else {
		ctxlog.Info(ctx, "taskRun already exists", namespace, request.Namespace, name, request.Name)

		err = r.GetBuildRunObject(ctx, lastTaskRun.Labels[buildv1alpha1.LabelBuildRun], request.Namespace, buildRun)
		if err != nil && !apierrors.IsNotFound(err) {
			return reconcile.Result{}, err
		} else if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		// Check if the BuildRun is already finished, this happens if the build controller is restarted.
		// It then reconciles all TaskRuns. This is valuable if the build controller was down while the TaskRun
		// finishes which would be missed otherwise. But, if the TaskRun was already completed and the status
		// synchronized into the BuildRun, then yet another reconciliation is not necessary.
		if buildRun.Status.CompletionTime != nil {
			ctxlog.Info(ctx, "buildRun already marked completed", namespace, request.Namespace, name, request.Name)
			return reconcile.Result{}, nil
		}

		trCondition := lastTaskRun.Status.GetCondition(apis.ConditionSucceeded)
		if trCondition != nil {
			if err := resources.UpdateBuildRunUsingTaskRunCondition(ctx, r.client, buildRun, lastTaskRun, trCondition); err != nil {
				return reconcile.Result{}, err
			}

			taskRunStatus := trCondition.Status

			// check if we should delete the generated service account by checking the build run spec and that the task run is csomplete
			if resources.IsGeneratedServiceAccountUsed(buildRun) && (taskRunStatus == corev1.ConditionTrue || taskRunStatus == corev1.ConditionFalse) {
				serviceAccount := &corev1.ServiceAccount{}
				serviceAccount.Name = resources.GetGeneratedServiceAccountName(buildRun)
				serviceAccount.Namespace = buildRun.Namespace

				ctxlog.Info(ctx, "deleting service account", namespace, request.Namespace, name, request.Name)
				if err = r.client.Delete(ctx, serviceAccount); err != nil && !apierrors.IsNotFound(err) {
					ctxlog.Error(ctx, err, "Error during deletion of generated service account.")
					return reconcile.Result{}, err
				}
			}

			buildRun.Status.Succeeded = taskRunStatus
			if taskRunStatus == corev1.ConditionFalse {
				buildRun.Status.Reason = trCondition.Message
			} else {
				buildRun.Status.Reason = trCondition.Reason
			}

			buildRun.Status.LatestTaskRunRef = &lastTaskRun.Name

			if buildRun.Status.StartTime == nil && lastTaskRun.Status.StartTime != nil {
				buildRun.Status.StartTime = lastTaskRun.Status.StartTime

				// Report the buildrun established duration (time between the creation of the buildrun and the start of the buildrun)
				buildmetrics.BuildRunEstablishObserve(
					buildRun.Status.BuildSpec.StrategyRef.Name,
					buildRun.Namespace,
					buildRun.Spec.BuildRef.Name,
					buildRun.Name,
					buildRun.Status.StartTime.Time.Sub(buildRun.CreationTimestamp.Time),
				)
			}

			if lastTaskRun.Status.CompletionTime != nil && buildRun.Status.CompletionTime == nil {
				buildRun.Status.CompletionTime = lastTaskRun.Status.CompletionTime

				if buildRun.Status.BuildSpec.StrategyRef != nil {
					// buildrun completion duration (total time between the creation of the buildrun and the buildrun completion)
					buildmetrics.BuildRunCompletionObserve(
						buildRun.Status.BuildSpec.StrategyRef.Name,
						buildRun.Namespace,
						buildRun.Spec.BuildRef.Name,
						buildRun.Name,
						buildRun.Status.CompletionTime.Time.Sub(buildRun.CreationTimestamp.Time),
					)

					// Look for the pod created by the taskrun
					var pod = &corev1.Pod{}
					if err := r.client.Get(ctx, types.NamespacedName{Namespace: request.Namespace, Name: lastTaskRun.Status.PodName}, pod); err == nil {
						if len(pod.Status.InitContainerStatuses) > 0 {

							lastInitPodIdx := len(pod.Status.InitContainerStatuses) - 1
							lastInitPod := pod.Status.InitContainerStatuses[lastInitPodIdx]

							if lastInitPod.State.Terminated != nil {
								// taskrun pod ramp-up (time between pod creation and last init container completion)
								buildmetrics.TaskRunPodRampUpDurationObserve(
									buildRun.Status.BuildSpec.StrategyRef.Name,
									buildRun.Namespace,
									buildRun.Spec.BuildRef.Name,
									buildRun.Name,
									lastInitPod.State.Terminated.FinishedAt.Sub(pod.CreationTimestamp.Time),
								)
							}
						}

						// taskrun ramp-up duration (time between taskrun creation and taskrun pod creation)
						buildmetrics.TaskRunRampUpDurationObserve(
							buildRun.Status.BuildSpec.StrategyRef.Name,
							buildRun.Namespace,
							buildRun.Spec.BuildRef.Name,
							buildRun.Name,
							pod.CreationTimestamp.Time.Sub(lastTaskRun.CreationTimestamp.Time),
						)

					}
				}
			}

			ctxlog.Info(ctx, "updating buildRun status", namespace, request.Namespace, name, request.Name)
			if err = r.client.Status().Update(ctx, buildRun); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	ctxlog.Debug(ctx, "finishing reconciling request from a BuildRun or TaskRun event", namespace, request.Namespace, name, request.Name)

	return reconcile.Result{}, nil
}

// GetBuildRunObject retrieves an existing BuildRun based on a name and namespace
func (r *ReconcileBuildRun) GetBuildRunObject(ctx context.Context, objectName string, objectNS string, buildRun *buildv1alpha1.BuildRun) error {
	if err := r.client.Get(ctx, types.NamespacedName{Name: objectName, Namespace: objectNS}, buildRun); err != nil {
		return err
	}
	return nil
}

// VerifyRequestName parse a Reconcile request name and looks for an associated BuildRun name
// If the BuildRun object exists and is not yet completed, it will update it with an error.
func (r *ReconcileBuildRun) VerifyRequestName(ctx context.Context, request reconcile.Request, buildRun *buildv1alpha1.BuildRun) {

	regxBuildRun, _ := regexp.Compile(generatedNameRegex)

	// Check if the name belongs to a TaskRun generated name https://regex101.com/r/Wjs3bV/10
	// and extract the BuildRun name
	matched := regxBuildRun.MatchString(request.Name)
	if matched {
		if split := regxBuildRun.Split(request.Name, 2); len(split) > 0 {
			// Update the related BuildRun
			err := r.GetBuildRunObject(ctx, split[0], request.Namespace, buildRun)
			if err == nil && buildRun.Status.CompletionTime == nil {
				// We ignore the errors from the following call, because the parent call of this function will always
				// return back a reconcile.Result{}, nil. This is done to avoid infinite reconcile loops when a BuildRun
				// does not longer exists
				r.updateBuildRunErrorStatus(ctx, buildRun, fmt.Sprintf("taskRun %s doesn't exist", request.Name))
			}
		}
	}
}

func (r *ReconcileBuildRun) createTaskRun(ctx context.Context, build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun) (*v1beta1.TaskRun, error) {
	var generatedTaskRun *v1beta1.TaskRun
	// Choose a service account to use
	serviceAccount, err := resources.RetrieveServiceAccount(ctx, r.client, build, buildRun)
	if err != nil {
		updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
		return nil, resources.HandleError("Failed to choose a service account to use", err, updateErr)
	}

	if build.Spec.StrategyRef.Kind == nil || *build.Spec.StrategyRef.Kind == buildv1alpha1.NamespacedBuildStrategyKind {
		buildStrategy, err := resources.RetrieveBuildStrategy(ctx, r.client, build)
		if err != nil {
			return nil, err
		}
		if buildStrategy != nil {
			generatedTaskRun, err = resources.GenerateTaskRun(r.config, build, buildRun, serviceAccount.Name, buildStrategy)
			if err != nil {
				updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
				return nil, resources.HandleError("Failed to generate the taskrun with buildStrategy", err, updateErr)
			}
		}
	} else if *build.Spec.StrategyRef.Kind == buildv1alpha1.ClusterBuildStrategyKind {
		clusterBuildStrategy, err := resources.RetrieveClusterBuildStrategy(ctx, r.client, build)
		if err != nil {
			return nil, err
		}
		if clusterBuildStrategy != nil {
			generatedTaskRun, err = resources.GenerateTaskRun(r.config, build, buildRun, serviceAccount.Name, clusterBuildStrategy)
			if err != nil {
				updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
				return nil, resources.HandleError("Failed to generate the taskrun with clusterBuildStrategy", err, updateErr)
			}
		}
	} else {
		err := fmt.Errorf("unknown strategy %s", string(*build.Spec.StrategyRef.Kind))
		updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
		return nil, resources.HandleError(fmt.Sprintf("Unsupported BuildStrategy Kind: %v", build.Spec.StrategyRef.Kind), err, updateErr)
	}

	// Set OwnerReference for BuildRun and TaskRun
	if err := r.setOwnerReferenceFunc(buildRun, generatedTaskRun, r.scheme); err != nil {
		updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
		return nil, resources.HandleError("failed to set OwnerReference for BuildRun and TaskRun", err, updateErr)
	}

	return generatedTaskRun, nil
}

// updateBuildRunErrorStatus updates buildRun status fields
func (r *ReconcileBuildRun) updateBuildRunErrorStatus(ctx context.Context, buildRun *buildv1alpha1.BuildRun, errorMessage string) error {
	// these two fields are deprecated and will be removed soon
	buildRun.Status.Succeeded = corev1.ConditionFalse
	buildRun.Status.Reason = errorMessage

	now := metav1.Now()
	buildRun.Status.CompletionTime = &now
	buildRun.Status.SetCondition(&buildv1alpha1.Condition{
		LastTransitionTime: now,
		Type:               buildv1alpha1.Succeeded,
		Status:             corev1.ConditionFalse,
		Reason:             "Failed",
		Message:            errorMessage,
	})
	ctxlog.Debug(ctx, "updating buildRun status", namespace, buildRun.Namespace, name, buildRun.Name)
	updateErr := r.client.Status().Update(ctx, buildRun)
	return updateErr
}

// ValidateBuildRegistration verifies that a referenced Build is properly registered
func (r *ReconcileBuildRun) ValidateBuildRegistration(ctx context.Context, build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun) error {
	if build.Status.Registered == "" {
		err := fmt.Errorf("the Build is not yet validated, build: %s", build.Name)
		return err
	}
	if build.Status.Registered != corev1.ConditionTrue {
		err := fmt.Errorf("the Build is not registered correctly, build: %s, registered status: %s, reason: %s", build.Name, build.Status.Registered, build.Status.Reason)
		updateErr := r.updateBuildRunErrorStatus(ctx, buildRun, err.Error())
		return resources.HandleError("Build is not ready", err, updateErr)
	}
	return nil
}

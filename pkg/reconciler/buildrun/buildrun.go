// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	buildmetrics "github.com/shipwright-io/build/pkg/metrics"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/runner"
	"github.com/shipwright-io/build/pkg/validate"
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
	config                *config.Config
	client                client.Client
	scheme                *runtime.Scheme
	setOwnerReferenceFunc setOwnerReferenceFunc
	runnerFactory         runner.BuildRunnerFactory
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(c *config.Config, mgr manager.Manager, ownerRef setOwnerReferenceFunc) reconcile.Reconciler {
	runnerFactory := runner.NewTaskRunBuildRunnerFactory(mgr.GetScheme(), c)
	return &ReconcileBuildRun{
		config:                c,
		client:                mgr.GetClient(),
		scheme:                mgr.GetScheme(),
		setOwnerReferenceFunc: ownerRef,
		runnerFactory:         runnerFactory,
	}
}

// Reconcile reads that state of the cluster for a Build object and makes changes based on the state read
// and what is in the Build.Spec
func (r *ReconcileBuildRun) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	var buildRun *buildv1beta1.BuildRun
	var build *buildv1beta1.Build

	updateBuildRunRequired := false

	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Debug(ctx, "starting reconciling request from a BuildRun or TaskRun event", namespace, request.Namespace, name, request.Name)

	// with build run cancel, it is now possible for a build run update to stem from something other than a task run update,
	// so we can no longer assume that a build run event will not come in after the build run has a task run ref in its status
	buildRun = &buildv1beta1.BuildRun{}
	getBuildRunErr := r.GetBuildRunObject(ctx, request.Name, request.Namespace, buildRun)
	lastTaskRun := &pipelineapi.TaskRun{}
	getTaskRunErr := r.client.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, lastTaskRun)

	if getBuildRunErr != nil && getTaskRunErr != nil {
		if !apierrors.IsNotFound(getBuildRunErr) {
			return reconcile.Result{}, getBuildRunErr
		}
		if !apierrors.IsNotFound(getTaskRunErr) {
			return reconcile.Result{}, getTaskRunErr
		}
		// If the BuildRun and TaskRun are not found, it might mean that we are running a Reconcile after a TaskRun was deleted. If this is the case, we need
		// to identify from the request the BuildRun name associate to it and update the BuildRun Status.
		r.VerifyRequestName(ctx, request, buildRun)
		return reconcile.Result{}, nil
	}

	// Skip validation in case buildrun could not be found, otherwise validate it
	if getBuildRunErr == nil {
		// Validating buildrun name is a valid label value
		if errs := validation.IsValidLabelValue(buildRun.Name); len(errs) > 0 {
			// stop reconciling and mark the BuildRun as Failed
			return reconcile.Result{}, resources.UpdateConditionWithFalseStatus(
				ctx,
				r.client,
				buildRun,
				strings.Join(errs, ", "),
				resources.BuildRunNameInvalid,
			)
		}

		// Validate BuildRun for disallowed field combinations (could technically be also done in a validating webhook)
		if reason, message := validate.BuildRunFields(buildRun); reason != "" {
			return reconcile.Result{}, resources.UpdateConditionWithFalseStatus(
				ctx,
				r.client,
				buildRun,
				message,
				reason,
			)
		}
	}

	// if this is a build run event after we've set the task run ref, get the task run using the task run name stored in the build run
	if getBuildRunErr == nil && apierrors.IsNotFound(getTaskRunErr) && buildRun.Status.TaskRunName != nil {
		getTaskRunErr = r.client.Get(ctx, types.NamespacedName{Name: *buildRun.Status.TaskRunName, Namespace: request.Namespace}, lastTaskRun)
	}

	// for existing TaskRuns update the BuildRun Status, if there is no TaskRun, then create one
	if getTaskRunErr != nil {
		if apierrors.IsNotFound(getTaskRunErr) {
			build = &buildv1beta1.Build{}
			if err := resources.GetBuildObject(ctx, r.client, buildRun, build); err != nil {
				if !resources.IsClientStatusUpdateError(err) && buildRun.Status.IsFailed(buildv1beta1.Succeeded) {
					return reconcile.Result{}, nil
				}
				// system call failure, reconcile again
				return reconcile.Result{}, err
			}

			// Validate if the Build was successfully registered
			if build.Status.Registered == nil || *build.Status.Registered == "" {
				switch {
				// When the build is referenced by name, it means the build is
				// an actual resource in the cluster and _should_ have been
				// validated and registered by now ...
				// reconcile again until it gets a registration value
				case buildRun.Spec.Build.Name != nil:
					return reconcile.Result{}, fmt.Errorf("the Build is not yet validated, build: %s", build.Name)

				// When the build(spec) is embedded in the buildrun, the now
				// transient/volatile build resource needs to be validated first
				case buildRun.Spec.Build.Spec != nil:
					err := validate.All(ctx,
						validate.NewSourceURL(r.client, build),
						validate.NewCredentials(r.client, build),
						validate.NewStrategies(r.client, build),
						validate.NewSourceRef(build),
						validate.NewBuildName(build),
						validate.NewEnv(build),
						validate.NewNodeSelector(build),
						validate.NewTolerations(build),
						validate.NewSchedulerName(build),
					)

					// an internal/technical error during validation happened
					if err != nil {
						return reconcile.Result{}, err
					}

					// one or more of the validations failed
					if build.Status.Reason != nil {
						return reconcile.Result{},
							resources.UpdateConditionWithFalseStatus(
								ctx,
								r.client,
								buildRun,
								*build.Status.Message,
								resources.ConditionBuildRegistrationFailed,
							)
					}
					// mark transient build as "registered" and validated
					build.Status.Registered = ptr.To(corev1.ConditionTrue)
					build.Status.Reason = ptr.To(buildv1beta1.SucceedStatus)
					build.Status.Message = ptr.To(buildv1beta1.AllValidationsSucceeded)
				}
			}

			if *build.Status.Registered != corev1.ConditionTrue {
				// stop reconciling and mark the BuildRun as Failed
				// we only reconcile again if the status.Update call fails
				var reason buildv1beta1.BuildReason

				if build.Status.Reason != nil {
					reason = *build.Status.Reason
				}

				message := fmt.Sprintf("the Build is not registered correctly, build: %s, registered status: %s, reason: %s", build.Name, *build.Status.Registered, reason)
				if updateErr := resources.UpdateConditionWithFalseStatus(ctx, r.client, buildRun, message, resources.ConditionBuildRegistrationFailed); updateErr != nil {
					return reconcile.Result{}, updateErr
				}

				return reconcile.Result{}, nil
			}

			// Ensure the build-related labels on the BuildRun
			if buildRun.GetLabels() == nil {
				buildRun.Labels = make(map[string]string)
			}

			// make sure the BuildRun has not already been cancelled
			if buildRun.IsCanceled() {
				if updateErr := resources.UpdateConditionWithFalseStatus(ctx, r.client, buildRun, "the BuildRun is marked canceled.", buildv1beta1.BuildRunStateCancel); updateErr != nil {
					return reconcile.Result{}, updateErr
				}
				return reconcile.Result{}, nil
			}

			// Set OwnerReference for Build and BuildRun only when build retention AtBuildDeletion is set to "true"
			if build.Spec.Retention != nil && build.Spec.Retention.AtBuildDeletion != nil {
				if *build.Spec.Retention.AtBuildDeletion && !resources.IsOwnedByBuild(build, buildRun.OwnerReferences) {
					if err := r.setOwnerReferenceFunc(build, buildRun, r.scheme); err != nil {
						build.Status.Reason = ptr.To(buildv1beta1.SetOwnerReferenceFailed)
						build.Status.Message = ptr.To(fmt.Sprintf("unexpected error when trying to set the ownerreference: %v", err))
						if err := r.client.Status().Update(ctx, build); err != nil {
							return reconcile.Result{}, err
						}
					}
					ctxlog.Info(ctx, fmt.Sprintf("updating BuildRun %s OwnerReferences, owner is Build %s", buildRun.Name, build.Name), namespace, request.Namespace, name, request.Name)
					updateBuildRunRequired = true
				}
			}

			// Add missing build name and generation labels to BuildRun (unless it is an embedded build)
			if build.Name != "" && build.Generation != 0 {
				buildGeneration := strconv.FormatInt(build.Generation, 10)
				if buildRun.GetLabels()[buildv1beta1.LabelBuild] != build.Name || buildRun.GetLabels()[buildv1beta1.LabelBuildGeneration] != buildGeneration {
					buildRun.Labels[buildv1beta1.LabelBuild] = build.Name
					buildRun.Labels[buildv1beta1.LabelBuildGeneration] = buildGeneration
					ctxlog.Info(ctx, "updating BuildRun labels", namespace, request.Namespace, name, request.Name)
					updateBuildRunRequired = true
				}
			}

			if updateBuildRunRequired {
				if err := r.client.Update(ctx, buildRun); err != nil {
					return reconcile.Result{}, err
				}
				ctxlog.Info(ctx, fmt.Sprintf("successfully updated BuildRun %s", buildRun.Name), namespace, request.Namespace, name, request.Name)
			}

			// Set the Build spec in the BuildRun status
			buildRun.Status.BuildSpec = &build.Spec
			ctxlog.Info(ctx, "updating BuildRun status", namespace, request.Namespace, name, request.Name)
			if err := r.client.Status().Update(ctx, buildRun); err != nil {
				return reconcile.Result{}, err
			}
			// Choose a service account to use
			svcAccount, err := resources.RetrieveServiceAccount(ctx, r.client, build, buildRun)
			if err != nil {
				if !resources.IsClientStatusUpdateError(err) && buildRun.Status.IsFailed(buildv1beta1.Succeeded) {
					return reconcile.Result{}, nil
				}
				// system call failure, reconcile again
				return reconcile.Result{}, err
			}

			strategy, err := r.getReferencedStrategy(ctx, build, buildRun)
			if err != nil {
				if !resources.IsClientStatusUpdateError(err) && buildRun.Status.IsFailed(buildv1beta1.Succeeded) {
					return reconcile.Result{}, nil
				}
				return reconcile.Result{}, err
			}

			// Validate the parameters
			valid, reason, message := validate.BuildRunParameters(strategy.GetParameters(), build.Spec.ParamValues, buildRun.Spec.ParamValues)
			if !valid {
				if err := resources.UpdateConditionWithFalseStatus(ctx, r.client, buildRun, message, reason); err != nil {
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, nil
			}

			// Validate the volumes
			valid, reason, message = validate.BuildRunVolumes(strategy.GetVolumes(), buildRun.Spec.Volumes)
			if !valid {
				if err := resources.UpdateConditionWithFalseStatus(ctx, r.client, buildRun, message, reason); err != nil {
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, nil
			}

			// Validate the nodeSelector
			valid, reason, message = validate.BuildRunNodeSelector(buildRun.Spec.NodeSelector)
			if !valid {
				if err := resources.UpdateConditionWithFalseStatus(ctx, r.client, buildRun, message, reason); err != nil {
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, nil
			}

			// Validate the tolerations
			valid, reason, message = validate.BuildRunTolerations(buildRun.Spec.Tolerations)
			if !valid {
				if err := resources.UpdateConditionWithFalseStatus(ctx, r.client, buildRun, message, reason); err != nil {
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, nil
			}

			// Validate the schedulerName
			valid, reason, message = validate.BuildRunSchedulerName(buildRun.Spec.SchedulerName)
			if !valid {
				if err := resources.UpdateConditionWithFalseStatus(ctx, r.client, buildRun, message, reason); err != nil {
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, nil
			}

			// This creates the TaskRun, but as an abstraced "BuildRunner" type
			// This will let us use other types in the future (ex: PipelineRun)
			buildRunner, err := r.runnerFactory.CreateBuildRunner(svcAccount, strategy, build, buildRun)
			if err != nil {
				if updateErr := resources.UpdateConditionWithFalseStatus(ctx, r.client, buildRun, err.Error(), resources.ConditionTaskRunGenerationFailed); updateErr != nil {
					err = resources.HandleError("failed to create taskrun runtime object", err, updateErr)
				}

				if !resources.IsClientStatusUpdateError(err) && buildRun.Status.IsFailed(buildv1beta1.Succeeded) {
					ctxlog.Info(ctx, "taskRun generation failed", namespace, request.Namespace, name, request.Name)
					return reconcile.Result{}, nil
				}
				// system call failure, reconcile again
				return reconcile.Result{}, err
			}

			validateErr := buildRunner.Validate(ctx, r.client)
			if validateErr != nil {
				// If the validation error is terminal, mark the build as failed.
				if validateErr.Terminal {
					if err := resources.UpdateConditionWithFalseStatus(ctx, r.client, buildRun, validateErr.Message, validateErr.ReasonCode); err != nil {
						return reconcile.Result{}, err
					}

					// end of reconciliation
					return reconcile.Result{}, nil
				}

				// some other error might have happened, return it and reconcile again
				return reconcile.Result{}, validateErr
			}
			ctxlog.Info(ctx, "creating TaskRun from BuildRun", namespace, request.Namespace, name, buildRunner.GetObject().GetGenerateName(), "BuildRun", buildRun.Name)
			if err = r.client.Create(ctx, buildRunner.GetObject()); err != nil {
				// system call failure, reconcile again
				return reconcile.Result{}, err
			}

			// Set the LastTaskRunRef in the BuildRun status
			runnerObjName := buildRunner.GetObject().GetName()
			buildRun.Status.TaskRunName = &runnerObjName
			ctxlog.Info(ctx, "updating BuildRun status with TaskRun name", namespace, request.Namespace, name, request.Name, "TaskRun", buildRunner.GetObject().GetName())
			if err = r.client.Status().Update(ctx, buildRun); err != nil {
				// we ignore the error here to prevent another reconciliation that would create another TaskRun,
				// the LatestTaskRunRef field will also be set in the reconciliation from a TaskRun
				// risk is that when the controller is now restarted before the field is set, another TaskRun will be created
				ctxlog.Error(ctx, err, "Failed to update BuildRun status is ignored", namespace, request.Namespace, name, request.Name)
			}

			// Increase BuildRun count in metrics
			buildmetrics.BuildRunCountInc(
				buildRun.Status.BuildSpec.StrategyName(),
				buildRun.Namespace,
				buildRun.Spec.BuildName(),
				buildRun.Name,
			)

			// Report buildrun ramp-up duration (time between buildrun creation and taskrun creation)
			buildmetrics.BuildRunRampUpDurationObserve(
				buildRun.Status.BuildSpec.StrategyName(),
				buildRun.Namespace,
				buildRun.Spec.BuildName(),
				buildRun.Name,
				buildRunner.GetObject().GetCreationTimestamp().Time.Sub(buildRun.CreationTimestamp.Time),
			)
		} else {
			return reconcile.Result{}, getTaskRunErr
		}
	} else {
		ctxlog.Info(ctx, "taskRun already exists", namespace, request.Namespace, name, request.Name)

		// Convert the TaskRun to our more abstract `BuildRunner` interface
		// In the future, the passed in object can use another Kubernetes type, such as a PipelineRun or even a plain Pod
		buildRunner, err := r.runnerFactory.ConvertToBuildRunner(lastTaskRun)
		if err != nil {
			return reconcile.Result{}, resources.UpdateConditionWithFalseStatus(ctx, r.client, buildRun, err.Error(), "InternalError")
		}

		if getBuildRunErr != nil && !apierrors.IsNotFound(getBuildRunErr) {
			return reconcile.Result{}, getBuildRunErr
		} else if apierrors.IsNotFound(getBuildRunErr) {
			// this is a TR event, try getting the br from the label on the tr
			err := r.GetBuildRunObject(ctx, buildRunner.GetObject().GetLabels()[buildv1beta1.LabelBuildRun], request.Namespace, buildRun)
			if err != nil && !apierrors.IsNotFound(err) {
				return reconcile.Result{}, err
			}
			if err != nil && apierrors.IsNotFound(err) {
				return reconcile.Result{}, nil
			}
		}

		if buildRun.IsCanceled() && !buildRunner.IsCancelled() {
			ctxlog.Info(ctx, "buildRun marked for cancellation, patching task run", namespace, request.Namespace, name, request.Name)

			if err := buildRunner.Cancel(ctx, r.client); err != nil {
				return reconcile.Result{}, err
			}
		}

		// Check if the BuildRun is already finished, this happens if the build controller is restarted.
		// It then reconciles all TaskRuns. This is valuable if the build controller was down while the TaskRun
		// finishes which would be missed otherwise. But, if the TaskRun was already completed and the status
		// synchronized into the BuildRun, then yet another reconciliation is not necessary.
		if buildRun.Status.CompletionTime != nil {
			ctxlog.Info(ctx, "buildRun already marked completed", namespace, request.Namespace, name, request.Name)
			return reconcile.Result{}, nil
		}

		// Update status of the BuildRun based on the underlying runner's state.
		// Note that this does an in-place update of the buildRun object - it doesn't issue the
		// client calls to update status.

		origStatus := buildRun.Status.DeepCopy()
		if err := buildRunner.SyncBuildRunStatus(ctx, r.client, buildRun); err != nil {
			return reconcile.Result{}, err
		}

		// check if we should delete the generated service account by checking the build run spec and that the task run is complete
		if buildRunner.IsCompleted() {
			if err := resources.DeleteServiceAccount(ctx, r.client, buildRun); err != nil {
				ctxlog.Error(ctx, err, "Error during deletion of generated service account.")
				return reconcile.Result{}, err
			}
		}
		objName := buildRunner.GetObject().GetName()
		buildRun.Status.TaskRunName = &objName

		if origStatus.StartTime == nil && buildRun.Status.StartTime != nil {
			// Report the buildrun established duration (time between the creation of the buildrun and the start of the buildrun)
			buildmetrics.BuildRunEstablishObserve(
				buildRun.Status.BuildSpec.StrategyName(),
				buildRun.Namespace,
				buildRun.Spec.BuildName(),
				buildRun.Name,
				buildRun.Status.StartTime.Time.Sub(buildRun.CreationTimestamp.Time),
			)
		}

		if origStatus.CompletionTime == nil && buildRun.Status.CompletionTime != nil {

			// buildrun completion duration (total time between the creation of the buildrun and the buildrun completion)
			buildmetrics.BuildRunCompletionObserve(
				buildRun.Status.BuildSpec.StrategyName(),
				buildRun.Namespace,
				buildRun.Spec.BuildName(),
				buildRun.Name,
				buildRun.Status.CompletionTime.Time.Sub(buildRun.CreationTimestamp.Time),
			)

			podCreationTime := buildRunner.GetPodCreationTime(ctx, r.client)
			podRampUpTime := buildRunner.GetPodInitFinishedTime(ctx, r.client)

			if podCreationTime != nil {
				// taskrun ramp-up duration (time between taskrun creation and taskrun pod creation)
				buildmetrics.TaskRunRampUpDurationObserve(
					buildRun.Status.BuildSpec.StrategyName(),
					buildRun.Namespace,
					buildRun.Spec.BuildName(),
					buildRun.Name,
					podCreationTime.Sub(buildRunner.GetCreationTimestamp().Time),
				)

				if podRampUpTime != nil {
					// taskrun pod ramp-up (time between pod creation and last init container completion)
					buildmetrics.TaskRunPodRampUpDurationObserve(
						buildRun.Status.BuildSpec.StrategyName(),
						buildRun.Namespace,
						buildRun.Spec.BuildName(),
						buildRun.Name,
						podRampUpTime.Sub(podCreationTime.Time),
					)
				}
			}
		}

		ctxlog.Info(ctx, "updating buildRun status", namespace, request.Namespace, name, request.Name)
		if err := r.client.Status().Update(ctx, buildRun); err != nil {
			return reconcile.Result{}, err
		}

	}

	ctxlog.Debug(ctx, "finishing reconciling request from a BuildRun or TaskRun event", namespace, request.Namespace, name, request.Name)

	return reconcile.Result{}, nil
}

// GetBuildRunObject retrieves an existing BuildRun based on a name and namespace
func (r *ReconcileBuildRun) GetBuildRunObject(ctx context.Context, objectName string, objectNS string, buildRun *buildv1beta1.BuildRun) error {
	return r.client.Get(ctx, types.NamespacedName{Name: objectName, Namespace: objectNS}, buildRun)
}

// VerifyRequestName parse a Reconcile request name and looks for an associated BuildRun name
// If the BuildRun object exists and is not yet completed, it will update it with an error.
func (r *ReconcileBuildRun) VerifyRequestName(ctx context.Context, request reconcile.Request, buildRun *buildv1beta1.BuildRun) {

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
				_ = resources.UpdateConditionWithFalseStatus(ctx, r.client, buildRun, fmt.Sprintf("taskRun %s doesn't exist", request.Name), resources.ConditionTaskRunIsMissing)
			}
		}
	}
}

func (r *ReconcileBuildRun) getReferencedStrategy(ctx context.Context, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) (strategy buildv1beta1.BuilderStrategy, err error) {
	if build.Spec.Strategy.Kind == nil {
		// If the strategy Kind is not specified, we default to a namespaced-scope strategy
		ctxlog.Info(ctx, "missing strategy Kind, defaulting to a namespaced-scope one", buildRun.Name, build.Name, namespace)
		strategy, err = resources.RetrieveBuildStrategy(ctx, r.client, build)
		if err != nil {
			if apierrors.IsNotFound(err) {
				if updateErr := resources.UpdateConditionWithFalseStatus(ctx, r.client, buildRun, err.Error(), resources.BuildStrategyNotFound); updateErr != nil {
					return nil, resources.HandleError("failed to get referenced strategy", err, updateErr)
				}
			}
		}
		return strategy, err
	}

	switch *build.Spec.Strategy.Kind {
	case buildv1beta1.NamespacedBuildStrategyKind:
		strategy, err = resources.RetrieveBuildStrategy(ctx, r.client, build)
		if err != nil {
			if apierrors.IsNotFound(err) {
				if updateErr := resources.UpdateConditionWithFalseStatus(ctx, r.client, buildRun, err.Error(), resources.BuildStrategyNotFound); updateErr != nil {
					return nil, resources.HandleError("failed to get referenced strategy", err, updateErr)
				}
			}
		}
	case buildv1beta1.ClusterBuildStrategyKind:
		strategy, err = resources.RetrieveClusterBuildStrategy(ctx, r.client, build)
		if err != nil {
			if apierrors.IsNotFound(err) {
				if updateErr := resources.UpdateConditionWithFalseStatus(ctx, r.client, buildRun, err.Error(), resources.ClusterBuildStrategyNotFound); updateErr != nil {
					return nil, resources.HandleError("failed to get referenced strategy", err, updateErr)
				}
			}
		}
	default:
		err = fmt.Errorf("unknown strategy %s", string(*build.Spec.Strategy.Kind))
		if updateErr := resources.UpdateConditionWithFalseStatus(ctx, r.client, buildRun, err.Error(), resources.ConditionUnknownStrategyKind); updateErr != nil {
			return nil, resources.HandleError("failed to get referenced strategy", err, updateErr)
		}
	}

	return strategy, err
}

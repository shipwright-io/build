// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/result"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/sources"
)

// TODO: Document these constants (as an API somehow?)
const (
	resultErrorMessage         = "error-message"
	resultErrorReason          = "error-reason"
	prefixedResultErrorReason  = prefixParamsResultsVolumes + "-" + resultErrorReason
	prefixedResultErrorMessage = prefixParamsResultsVolumes + "-" + resultErrorMessage
)

type TaskRunBuildRunner struct {
	*pipelineapi.TaskRun
}

func (t *TaskRunBuildRunner) GetCompletionTime() *metav1.Time {
	return t.Status.CompletionTime
}

func (t *TaskRunBuildRunner) GetStartTime() *metav1.Time {
	return t.Status.StartTime
}

func (t *TaskRunBuildRunner) GetObject() client.Object {
	return t.TaskRun
}

func (t *TaskRunBuildRunner) GetPodCreationTime(ctx context.Context, client client.Client) *metav1.Time {
	pod := t.getTaskRunPod(ctx, client)
	if pod == nil {
		return nil
	}
	return &pod.CreationTimestamp
}

func (t *TaskRunBuildRunner) getTaskRunPod(ctx context.Context, client client.Client) *corev1.Pod {
	pod := &corev1.Pod{}
	if err := client.Get(ctx, types.NamespacedName{
		Namespace: t.TaskRun.GetNamespace(),
		Name:      t.TaskRun.Status.PodName}, pod); err != nil {
		return nil
	}
	return pod
}

func (t *TaskRunBuildRunner) GetPodInitFinishedTime(ctx context.Context, client client.Client) *metav1.Time {
	pod := t.getTaskRunPod(ctx, client)
	if pod == nil {
		return nil
	}
	initContainersCount := len(pod.Status.InitContainerStatuses)
	if initContainersCount == 0 {
		return nil
	}

	lastInitPod := pod.Status.InitContainerStatuses[initContainersCount-1]
	if lastInitPod.State.Terminated == nil {
		return nil
	}
	return &lastInitPod.State.Terminated.FinishedAt
}

func (t *TaskRunBuildRunner) IsCompleted() bool {
	succeededCondition := t.Status.GetCondition(apis.ConditionSucceeded)
	if succeededCondition == nil {
		return false
	}
	return succeededCondition.IsFalse() || succeededCondition.IsTrue()
}

func (t *TaskRunBuildRunner) Validate(ctx context.Context, client client.Client) *BuildRunnerValidationError {
	err := resources.CheckTaskRunVolumesExist(ctx, client, t.TaskRun)
	if err == nil {
		return nil
	}
	validationErr := &BuildRunnerValidationError{
		Message: err.Error(),
	}
	if apierrors.IsNotFound(err) {
		validationErr.Terminal = true
		validationErr.ReasonCode = string(v1beta1.VolumeDoesNotExist)
	}
	return validationErr
}

func (t *TaskRunBuildRunner) Cancel(ctx context.Context, client client.Client) error {
	// patch tekton taskrun a la tkn to start tekton's cancelling logic
	trueParam := true
	return t.patchTaskRun(ctx, client, t.TaskRun, "replace", "/spec/status", pipelineapi.TaskRunSpecStatusCancelled, metav1.PatchOptions{Force: &trueParam})
}

type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func (t *TaskRunBuildRunner) patchTaskRun(ctx context.Context, k8sClient client.Client, tr *pipelineapi.TaskRun, op, path, value string, opts metav1.PatchOptions) error {
	payload := []patchStringValue{{
		Op:    op,
		Path:  path,
		Value: value,
	}}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	patch := client.RawPatch(types.JSONPatchType, data)
	patchOpt := client.PatchOptions{Raw: &opts}
	return k8sClient.Patch(ctx, tr, patch, &patchOpt)
}

func (t *TaskRunBuildRunner) SyncBuildRunStatus(ctx context.Context, client client.Client, buildRun *v1beta1.BuildRun) error {

	if buildRun.Status.StartTime == nil && t.GetStartTime() != nil {
		buildRun.Status.StartTime = t.GetStartTime()
	}

	if buildRun.Status.CompletionTime == nil && t.GetCompletionTime() != nil {
		buildRun.Status.CompletionTime = t.GetCompletionTime()
	}

	if len(t.Status.Results) > 0 {
		// TODO: this logic could be optimized so we iterate through the result values only once.
		updateBuildRunStatusWithSourceResult(ctx, buildRun, t.Status.Results)
		updateBuildRunStatusWithOutputResult(ctx, buildRun, t.Status.Results)
	}

	succeededCondition := t.Status.GetCondition(apis.ConditionSucceeded)
	if succeededCondition == nil {
		return nil
	}

	if err := t.updateBuildRunUsingTaskRunCondition(ctx, client, buildRun, succeededCondition); err != nil {
		return err
	}

	t.updateBuildRunUsingTaskFailures(ctx, client, buildRun)

	return nil
}

const (
	defaultSourceName          = "default"
	sourceTimestampName        = "source-timestamp"
	prefixParamsResultsVolumes = "shp"
	imageDigestResult          = "image-digest"
	imageSizeResult            = "image-size"
	imageVulnerabilities       = "image-vulnerabilities"
)

func updateBuildRunStatusWithSourceResult(ctx context.Context, buildrun *v1beta1.BuildRun, results []pipelineapi.TaskRunResult) {
	buildSpec := buildrun.Status.BuildSpec

	if buildSpec.Source == nil {
		return
	}

	switch {
	case buildSpec.Source.Type == v1beta1.OCIArtifactType && buildSpec.Source.OCIArtifact != nil:
		sources.AppendBundleResult(buildrun, defaultSourceName, results)

	case buildSpec.Source.Type == v1beta1.GitType && buildSpec.Source.Git != nil:
		sources.AppendGitResult(buildrun, defaultSourceName, results)
	}

	if sourceTimestamp := sources.FindResultValue(results, defaultSourceName, sourceTimestampName); strings.TrimSpace(sourceTimestamp) != "" {
		sec, err := strconv.ParseInt(sourceTimestamp, 10, 64)
		if err != nil {
			ctxlog.Error(ctx, err, "failed to parse the source timestamp")
			return
		}
		if buildrun.Status.Source != nil {
			buildrun.Status.Source.Timestamp = &metav1.Time{Time: time.Unix(sec, 0)}
		}
	}
}

func updateBuildRunStatusWithOutputResult(ctx context.Context, buildRun *v1beta1.BuildRun, taskRunResult []pipelineapi.TaskRunResult) {
	if buildRun.Status.Output == nil {
		buildRun.Status.Output = &v1beta1.Output{}
	}

	for _, result := range taskRunResult {
		switch result.Name {
		case generateOutputResultName(imageDigestResult):
			buildRun.Status.Output.Digest = result.Value.StringVal

		case generateOutputResultName(imageSizeResult):
			size, err := strconv.ParseInt(result.Value.StringVal, 10, 64)
			if err != nil {
				ctxlog.Error(ctx, err, "failed to parse output image size")
				continue
			}
			buildRun.Status.Output.Size = size
		case generateOutputResultName(imageVulnerabilities):
			buildRun.Status.Output.Vulnerabilities = getImageVulnerabilitiesResult(result)
		}
	}
}

func generateOutputResultName(resultName string) string {
	return fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultName)
}

func getImageVulnerabilitiesResult(result pipelineapi.TaskRunResult) []v1beta1.Vulnerability {
	var vulns []v1beta1.Vulnerability
	if len(result.Value.StringVal) == 0 {
		return vulns
	}

	vulnerabilities := strings.Split(result.Value.StringVal, ",")
	for _, vulnerability := range vulnerabilities {
		vuln := strings.Split(vulnerability, ":")
		severity := getSeverity(vuln[1])
		vulns = append(vulns, v1beta1.Vulnerability{
			ID:       vuln[0],
			Severity: severity,
		})
	}
	return vulns
}

func getSeverity(sev string) v1beta1.VulnerabilitySeverity {
	switch strings.ToUpper(sev) {
	case "L":
		return v1beta1.Low
	case "M":
		return v1beta1.Medium
	case "H":
		return v1beta1.High
	case "C":
		return v1beta1.Critical
	default:
		return v1beta1.Unknown
	}
}

func (t *TaskRunBuildRunner) updateBuildRunUsingTaskRunCondition(ctx context.Context, client client.Client, buildRun *v1beta1.BuildRun, trCondition *apis.Condition) error {
	var reason, message string = trCondition.Reason, trCondition.Message
	status := trCondition.Status

	switch pipelineapi.TaskRunReason(reason) {
	case pipelineapi.TaskRunReasonStarted:
		fallthrough
	case pipelineapi.TaskRunReasonRunning:
		if buildRun.IsCanceled() {
			status = corev1.ConditionUnknown // in practice the taskrun status is already unknown in this case, but we are making sure here
			reason = v1beta1.BuildRunStateCancel
			message = "The user requested the BuildRun to be canceled.  This BuildRun controller has requested the TaskRun be canceled.  That request has not been process by Tekton's TaskRun controller yet."
		}
	case pipelineapi.TaskRunReasonCancelled:
		if buildRun.IsCanceled() {
			status = corev1.ConditionFalse // in practice the taskrun status is already false in this case, bue we are making sure here
			reason = v1beta1.BuildRunStateCancel
			message = "The BuildRun and underlying TaskRun were canceled successfully."
		}

	case pipelineapi.TaskRunReasonTimedOut:
		reason = "BuildRunTimeout"
		message = fmt.Sprintf("BuildRun %s failed to finish within %s",
			buildRun.Name,
			t.TaskRun.Spec.Timeout.Duration,
		)

	case pipelineapi.TaskRunReasonSuccessful:
		if buildRun.IsCanceled() {
			message = "The TaskRun completed before the request to cancel the TaskRun could be processed."
		}

	case pipelineapi.TaskRunReasonFailed:
		if t.TaskRun.Status.CompletionTime != nil {
			pod, failedContainer, failedContainerStatus, err := extractFailedPodAndContainer(ctx, client, t.TaskRun)
			if err != nil {
				// when trying to customize the Condition Message field, ensure the Message cover the case
				// when a Pod is deleted.
				// Note: this is an edge case, but not doing this prevent a BuildRun from being marked as Failed
				// while the TaskRun is already with a Failed Reason in it´s condition.
				if apierrors.IsNotFound(err) {
					message = fmt.Sprintf("buildrun failed, pod %s/%s not found", t.TaskRun.Namespace, t.TaskRun.Status.PodName)
					break
				}
				return err
			}

			//nolint:staticcheck // SA1019 we want to give users some time to adopt to failureDetails
			buildRun.Status.FailureDetails = &v1beta1.FailureDetails{
				Location: &v1beta1.Location{
					Pod: pod.Name,
				},
			}

			if pod.Status.Reason == "Evicted" {
				message = pod.Status.Message
				reason = v1beta1.BuildRunStatePodEvicted
				if failedContainer != nil {
					buildRun.Status.FailureDetails.Location.Container = failedContainer.Name
				}
			} else if failedContainer != nil {
				buildRun.Status.FailureDetails.Location.Container = failedContainer.Name

				message = fmt.Sprintf("buildrun step %s failed, for detailed information: kubectl --namespace %s logs %s --container=%s",
					failedContainer.Name,
					pod.Namespace,
					pod.Name,
					failedContainer.Name,
				)

				if failedContainerStatus != nil && failedContainerStatus.State.Terminated != nil {
					if failedContainerStatus.State.Terminated.Reason == "OOMKilled" {
						reason = v1beta1.BuildRunStateStepOutOfMemory
						message = fmt.Sprintf("buildrun step %s failed due to out-of-memory, for detailed information: kubectl --namespace %s logs %s --container=%s",
							failedContainer.Name,
							pod.Namespace,
							pod.Name,
							failedContainer.Name,
						)
					} else if failedContainer.Name == "step-image-processing" && failedContainerStatus.State.Terminated.ExitCode == 22 {
						reason = v1beta1.BuildRunStateVulnerabilitiesFound
						message = fmt.Sprintf("Vulnerabilities have been found in the image which can be seen in the buildrun status. For detailed information,see kubectl --namespace %s logs %s --container=%s",
							pod.Namespace,
							pod.Name,
							failedContainer.Name,
						)
					}
				}
			} else {
				message = fmt.Sprintf("buildrun failed due to an unexpected error in pod %s: for detailed information: kubectl --namespace %s logs %s --all-containers",
					pod.Name,
					pod.Namespace,
					pod.Name,
				)
			}
		}
	}

	buildRun.Status.SetCondition(&v1beta1.Condition{
		LastTransitionTime: metav1.Now(),
		Type:               v1beta1.Succeeded,
		Status:             status,
		Reason:             reason,
		Message:            message,
	})

	return nil
}

func extractFailureDetails(ctx context.Context, client client.Client, taskRun *pipelineapi.TaskRun) (failure *v1beta1.FailureDetails) {
	failure = &v1beta1.FailureDetails{}

	failure.Reason, failure.Message = extractFailureReasonAndMessage(taskRun)

	failure.Location = &v1beta1.Location{Pod: taskRun.Status.PodName}
	pod, container, _, _ := extractFailedPodAndContainer(ctx, client, taskRun)

	if pod != nil && container != nil {
		failure.Location.Pod = pod.Name
		failure.Location.Container = container.Name
	}

	return failure
}

func extractFailureReasonAndMessage(taskRun *pipelineapi.TaskRun) (errorReason string, errorMessage string) {
	for _, step := range taskRun.Status.Steps {
		if step.Terminated == nil || step.Terminated.ExitCode == 0 {
			continue
		}

		message := step.Terminated.Message
		var taskRunResults []result.RunResult

		if err := json.Unmarshal([]byte(message), &taskRunResults); err != nil {
			continue
		}

		for _, result := range taskRunResults {
			if result.Key == prefixedResultErrorMessage {
				errorMessage = result.Value
			}

			if result.Key == prefixedResultErrorReason {
				errorReason = result.Value
			}
		}
	}

	return errorReason, errorMessage
}

func extractFailedPodAndContainer(ctx context.Context, client client.Client, taskRun *pipelineapi.TaskRun) (*corev1.Pod, *corev1.Container, *corev1.ContainerStatus, error) {
	var pod corev1.Pod
	if err := client.Get(ctx, types.NamespacedName{Namespace: taskRun.Namespace, Name: taskRun.Status.PodName}, &pod); err != nil {
		return nil, nil, nil, err
	}

	failures := make(map[string]*corev1.ContainerStatus)
	// Find the names of all containers with failure status
	for i := range pod.Status.ContainerStatuses {
		containerStatus := pod.Status.ContainerStatuses[i]
		if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode != 0 {
			failures[containerStatus.Name] = &containerStatus
		}
	}

	// Find the first container that has a failure status
	var failedContainer *corev1.Container
	var failedContainerStatus *corev1.ContainerStatus
	for i, container := range pod.Spec.Containers {
		if containerStatus, has := failures[container.Name]; has {
			failedContainer = &pod.Spec.Containers[i]
			failedContainerStatus = containerStatus
			break
		}
	}

	return &pod, failedContainer, failedContainerStatus, nil
}

// updateBuildRunUsingTaskFailures is extracting failures from taskRun steps and adding them to buildRun (mutates)
func (t *TaskRunBuildRunner) updateBuildRunUsingTaskFailures(ctx context.Context, client client.Client, buildRun *v1beta1.BuildRun) {
	trCondition := t.TaskRun.Status.GetCondition(apis.ConditionSucceeded)

	// only extract failures when failing condition is present
	if trCondition != nil && pipelineapi.TaskRunReason(trCondition.Reason) == pipelineapi.TaskRunReasonFailed {
		buildRun.Status.FailureDetails = extractFailureDetails(ctx, client, t.TaskRun)
	}
}

type TaskRunBuildRunnerFactory struct {
	config *config.Config
	scheme *runtime.Scheme
}

// NewTaskRunBuildRunnerFactory creates a new TaskRunBuildRunnerFactory instance.
func NewTaskRunBuildRunnerFactory(scheme *runtime.Scheme, controllerConfig *config.Config) BuildRunnerFactory {
	return &TaskRunBuildRunnerFactory{
		scheme: scheme,
		config: controllerConfig,
	}
}

func (t *TaskRunBuildRunnerFactory) CreateBuildRunner(serviceAccount *corev1.ServiceAccount, strategy v1beta1.BuilderStrategy, build *v1beta1.Build, buildRun *v1beta1.BuildRun) (BuildRunner, error) {
	var (
		generatedTaskRun *pipelineapi.TaskRun
	)

	generatedTaskRun, err := resources.GenerateTaskRun(t.config, build, buildRun, serviceAccount.Name, strategy)

	if err != nil {
		return nil, err
	}

	// Set OwnerReference for BuildRun and TaskRun
	if err := controllerutil.SetOwnerReference(buildRun, generatedTaskRun, t.scheme); err != nil {
		return nil, err
	}

	return &TaskRunBuildRunner{
		TaskRun: generatedTaskRun,
	}, nil
}

// ConvertToBuildRunner converts the provided TaskRun into a BuildRunner instance.
func (t *TaskRunBuildRunnerFactory) ConvertToBuildRunner(obj client.Object) (BuildRunner, error) {
	switch runner := obj.(type) {
	case *pipelineapi.TaskRun:
		return &TaskRunBuildRunner{TaskRun: runner}, nil
	default:
		return nil, fmt.Errorf("cannot convert provided type to BuildRunner")
	}
}

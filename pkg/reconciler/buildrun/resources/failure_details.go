// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/ctxlog"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/result"
	"knative.dev/pkg/apis"
)

const (
	resultErrorMessage         = "error-message"
	resultErrorReason          = "error-reason"
	prefixedResultErrorReason  = prefixParamsResultsVolumes + "-" + resultErrorReason
	prefixedResultErrorMessage = prefixParamsResultsVolumes + "-" + resultErrorMessage
)

// UpdateBuildRunUsingTaskFailures is extracting failures from taskRun steps and adding them to buildRun (mutates)
func UpdateBuildRunUsingTaskFailures(ctx context.Context, client client.Client, buildRun *buildv1beta1.BuildRun, taskRun *pipelineapi.TaskRun) {
	trCondition := taskRun.Status.GetCondition(apis.ConditionSucceeded)

	// only extract failures when failing condition is present
	if trCondition != nil && pipelineapi.TaskRunReason(trCondition.Reason) == pipelineapi.TaskRunReasonFailed {
		buildRun.Status.FailureDetails = extractFailureDetails(ctx, client, taskRun)
	}
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

func extractFailedPodAndContainer(ctx context.Context, client client.Client, taskRun *pipelineapi.TaskRun) (*corev1.Pod, *corev1.Container, error) {
	var pod corev1.Pod
	if err := client.Get(ctx, types.NamespacedName{Namespace: taskRun.Namespace, Name: taskRun.Status.PodName}, &pod); err != nil {
		ctxlog.Error(ctx, err, "failed to get pod for failure extraction", namespace, taskRun.Namespace, name, taskRun.Status.PodName)
		return nil, nil, err
	}

	failures := make(map[string]struct{})
	// Find the names of all containers with failure status
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode != 0 {
			failures[containerStatus.Name] = struct{}{}
		}
	}

	// Find the first container that has a failure status
	var failedContainer *corev1.Container
	for i, container := range pod.Spec.Containers {
		if _, has := failures[container.Name]; has {
			failedContainer = &pod.Spec.Containers[i]
			break
		}
	}

	return &pod, failedContainer, nil
}

func extractFailureDetails(ctx context.Context, client client.Client, taskRun *pipelineapi.TaskRun) (failure *buildv1beta1.FailureDetails) {
	failure = &buildv1beta1.FailureDetails{}

	failure.Reason, failure.Message = extractFailureReasonAndMessage(taskRun)

	failure.Location = &buildv1beta1.Location{Pod: taskRun.Status.PodName}
	pod, container, _ := extractFailedPodAndContainer(ctx, client, taskRun)

	if pod != nil && container != nil {
		failure.Location.Pod = pod.Name
		failure.Location.Container = container.Name
	}

	return failure
}

func getFailureDetailsTaskSpecResults() []pipelineapi.TaskResult {
	return []pipelineapi.TaskResult{
		{
			Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorMessage),
			Description: "The error description of the task run",
		},
		{
			Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorReason),
			Description: "The error reason of the task run",
		},
	}
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	prefixedResultErrorReason  = prefixParamsResultsVolumes + "-" + resultErrorReason
	prefixedResultErrorMessage = prefixParamsResultsVolumes + "-" + resultErrorMessage
)

// UpdateBuildRunUsingTaskFailures is extracting failures from taskRun steps and adding them to buildRun (mutates)
func UpdateBuildRunUsingTaskFailures(ctx context.Context, client client.Client, buildRun *buildv1alpha1.BuildRun, taskRun *v1beta1.TaskRun) {
	trCondition := taskRun.Status.GetCondition(apis.ConditionSucceeded)

	// only extract failures when failing condition is present
	if trCondition != nil && v1beta1.TaskRunReason(trCondition.Reason) == v1beta1.TaskRunReasonFailed {
		buildRun.Status.FailureDetails = extractFailureDetails(ctx, client, taskRun)
	}
}

func extractFailureReasonAndMessage(taskRun *v1beta1.TaskRun) (errorReason string, errorMessage string, hasReasonAndMessage bool) {
	for _, step := range taskRun.Status.Steps {
		message := step.Terminated.Message
		var taskRunResults []v1beta1.PipelineResourceResult

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

	return errorReason, errorMessage, true
}

func extractFailedPodAndContainer(ctx context.Context, client client.Client, taskRun *v1beta1.TaskRun) (*v1.Pod, *v1.Container, error) {
	var pod v1.Pod
	if err := client.Get(ctx, types.NamespacedName{Namespace: taskRun.Namespace, Name: taskRun.Status.PodName}, &pod); err != nil {
		return nil, nil, err
	}

	failures := make(map[string]struct{})
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode != 0 {
			failures[containerStatus.Name] = struct{}{}
		}
	}

	// Find the first container that failed
	var failedContainer *v1.Container
	for i, container := range pod.Spec.Containers {
		if _, has := failures[container.Name]; has {
			failedContainer = &pod.Spec.Containers[i]
			break
		}
	}

	return &pod, failedContainer, nil
}

func extractFailureDetails(ctx context.Context, client client.Client, taskRun *v1beta1.TaskRun) (failure *buildv1alpha1.FailureDetails) {
	failure = &buildv1alpha1.FailureDetails{ }
	failure.Location = &buildv1alpha1.FailedAt{ Pod: taskRun.Status.PodName }

	if reason, message, hasReasonAndMessage := extractFailureReasonAndMessage(taskRun); hasReasonAndMessage {
		failure.Reason = reason
		failure.Message = message
	}

	pod, container, _ := extractFailedPodAndContainer(ctx, client, taskRun)

	if pod != nil && container != nil {
		failure.Location.Pod = pod.Name
		failure.Location.Container = container.Name
	}

	return failure
}

func getFailureDetailsTaskSpecResults() []v1beta1.TaskResult {
	return []v1beta1.TaskResult{
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

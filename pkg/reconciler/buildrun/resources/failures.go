package resources

import (
	"context"
	"encoding/json"
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

func extractFailureReasonAndMessage(taskRun *v1beta1.TaskRun) *buildv1alpha1.FailureDetails {
	shipError := buildv1alpha1.FailureDetails{}

	for _, step := range taskRun.Status.Steps {
		message := step.Terminated.Message
		var taskRunResults []v1beta1.PipelineResourceResult

		if err := json.Unmarshal([]byte(message), &taskRunResults); err != nil {
			continue
		}

		for _, result := range taskRunResults {
			if result.Key == prefixedResultErrorMessage {
				shipError.Message = result.Value
			}

			if result.Key == prefixedResultErrorReason {
				shipError.Reason = result.Value
			}
		}
	}

	if len(shipError.Message) == 0 || len(shipError.Reason) == 0 {
		return nil
	}

	return &shipError
}

func extractFailedPodAndContainer(ctx context.Context, client client.Client, taskRun *v1beta1.TaskRun) (*v1.Pod, *v1.Container, error) {
	var pod v1.Pod
	if err := client.Get(ctx, types.NamespacedName{Namespace: taskRun.Namespace, Name: taskRun.Status.PodName}, &pod); err != nil {
		return nil, nil, err
	}

	var failures = make(map[string]struct{})
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
	if failure = extractFailureReasonAndMessage(taskRun); failure == nil {
		return nil
	}

	pod, container, _ := extractFailedPodAndContainer(ctx, client, taskRun)

	if pod == nil || container == nil {
		return failure
	}

	failure.Location = &buildv1alpha1.FailedAt{Container: container.Name, Pod: pod.Name}

	return failure
}

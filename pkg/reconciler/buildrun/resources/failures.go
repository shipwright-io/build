package resources

import (
	"encoding/json"
	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/apis"
)

const (
	prefixedResultErrorReason  = prefixParamsResultsVolumes + "-" + resultErrorReason
	prefixedResultErrorMessage = prefixParamsResultsVolumes + "-" + resultErrorMessage
)

// UpdateBuildRunUsingTaskFailures is extracting failures from taskRun steps and adding them to buildRun (mutates)
func UpdateBuildRunUsingTaskFailures(buildRun *buildv1alpha1.BuildRun, taskRun *v1beta1.TaskRun) {
	trCondition := taskRun.Status.GetCondition(apis.ConditionSucceeded)

	// only extract failures when failing condition is present
	if trCondition != nil && v1beta1.TaskRunReason(trCondition.Reason) == v1beta1.TaskRunReasonFailed {
		buildRun.Status.Failure = extractErrorFromTaskRun(taskRun)
	}
}

func extractErrorFromTaskRun(taskRun *v1beta1.TaskRun) *buildv1alpha1.Failure {
	shipError := buildv1alpha1.Failure{}

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

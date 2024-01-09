// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/result"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	buildfakes "github.com/shipwright-io/build/pkg/controller/fakes"
)

var _ = Describe("Surfacing errors", func() {
	Context("resources.UpdateBuildRunUsingTaskFailures", func() {
		ctx := context.Background()
		client := &buildfakes.FakeClient{}

		It("surfaces errors to BuildRun in failed TaskRun", func() {
			redTaskRun := pipelineapi.TaskRun{}
			redTaskRun.Status.Conditions = append(redTaskRun.Status.Conditions,
				apis.Condition{Type: apis.ConditionSucceeded, Reason: pipelineapi.TaskRunReasonFailed.String()})
			failedStep := pipelineapi.StepState{}

			errorReasonValue := "PullBaseImageFailed"
			errorMessageValue := "Failed to pull the base image."
			errorReasonKey := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorReason)
			errorMessageKey := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorMessage)

			errorReason := result.RunResult{Key: errorReasonKey, Value: errorReasonValue}
			errorMessage := result.RunResult{Key: errorMessageKey, Value: errorMessageValue}
			unrelated := result.RunResult{Key: "unrelated-resource-key", Value: "Unrelated resource value"}

			message, _ := json.Marshal([]result.RunResult{errorReason, errorMessage, unrelated})

			failedStep.Terminated = &corev1.ContainerStateTerminated{Message: string(message), ExitCode: 1}
			followUpStep := pipelineapi.StepState{}

			redTaskRun.Status.Steps = append(redTaskRun.Status.Steps, failedStep, followUpStep)
			redBuild := buildv1beta1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(ctx, client, &redBuild, &redTaskRun)

			Expect(redBuild.Status.FailureDetails.Message).To(Equal(errorMessageValue))
			Expect(redBuild.Status.FailureDetails.Reason).To(Equal(errorReasonValue))
		})

		It("does not surface unrelated Tekton resources if the TaskRun fails", func() {
			redTaskRun := pipelineapi.TaskRun{}
			redTaskRun.Status.Conditions = append(redTaskRun.Status.Conditions,
				apis.Condition{Type: apis.ConditionSucceeded, Reason: pipelineapi.TaskRunReasonFailed.String()})
			failedStep := pipelineapi.StepState{}

			unrelated := result.RunResult{Key: "unrelated", Value: "unrelated"}

			message, _ := json.Marshal([]result.RunResult{unrelated})

			failedStep.Terminated = &corev1.ContainerStateTerminated{Message: string(message)}

			redTaskRun.Status.Steps = append(redTaskRun.Status.Steps, failedStep)
			redBuild := buildv1beta1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(ctx, client, &redBuild, &redTaskRun)

			Expect(redBuild.Status.FailureDetails.Reason).To(BeEmpty())
			Expect(redBuild.Status.FailureDetails.Message).To(BeEmpty())
		})

		It("does not surface error results if the container terminated without failure", func() {
			greenTaskRun := pipelineapi.TaskRun{}
			greenTaskRun.Status.Conditions = append(greenTaskRun.Status.Conditions,
				apis.Condition{Type: apis.ConditionSucceeded, Reason: pipelineapi.TaskRunReasonSuccessful.String()})
			failedStep := pipelineapi.StepState{}

			errorReasonValue := "PullBaseImageFailed"
			errorMessageValue := "Failed to pull the base image."
			errorReasonKey := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorReason)
			errorMessageKey := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorMessage)

			errorReason := result.RunResult{Key: errorReasonKey, Value: errorReasonValue}
			errorMessage := result.RunResult{Key: errorMessageKey, Value: errorMessageValue}
			message, _ := json.Marshal([]result.RunResult{errorReason, errorMessage})

			failedStep.Terminated = &corev1.ContainerStateTerminated{Message: string(message)}

			greenTaskRun.Status.Steps = append(greenTaskRun.Status.Steps, failedStep)
			greenBuildRun := buildv1beta1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(ctx, client, &greenBuildRun, &greenTaskRun)

			Expect(greenBuildRun.Status.FailureDetails).To(BeNil())
		})

		It("should not surface errors for a successful TaskRun", func() {
			greenTaskRun := pipelineapi.TaskRun{}
			greenTaskRun.Status.Conditions = append(greenTaskRun.Status.Conditions, apis.Condition{Type: apis.ConditionSucceeded})
			greenBuildRun := buildv1beta1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(ctx, client, &greenBuildRun, &greenTaskRun)

			Expect(greenBuildRun.Status.FailureDetails).To(BeNil())
		})

		It("should not surface errors if the TaskRun does not have a Succeeded condition", func() {
			unfinishedTaskRun := pipelineapi.TaskRun{}
			unfinishedTaskRun.Status.Conditions = append(unfinishedTaskRun.Status.Conditions, apis.Condition{Type: apis.ConditionReady})
			unfinishedBuildRun := buildv1beta1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(ctx, client, &unfinishedBuildRun, &unfinishedTaskRun)
			Expect(unfinishedBuildRun.Status.FailureDetails).To(BeNil())
		})

		It("should not surface errors if the TaskRun is in progress", func() {
			unknownTaskRun := pipelineapi.TaskRun{}
			unknownTaskRun.Status.Conditions = append(unknownTaskRun.Status.Conditions, apis.Condition{Type: apis.ConditionSucceeded, Reason: "random"})
			unknownBuildRun := buildv1beta1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(ctx, client, &unknownBuildRun, &unknownTaskRun)
			Expect(unknownBuildRun.Status.FailureDetails).To(BeNil())
		})
	})
})

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
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildfakes "github.com/shipwright-io/build/pkg/controller/fakes"
)

var _ = Describe("Surfacing errors", func() {
	Context("resources.UpdateBuildRunUsingTaskFailures", func() {
		ctx := context.Background()
		client := &buildfakes.FakeClient{}

		It("surfaces errors to BuildRun in failed TaskRun", func() {
			redTaskRun := pipelinev1beta1.TaskRun{}
			redTaskRun.Status.Conditions = append(redTaskRun.Status.Conditions,
				apis.Condition{Type: apis.ConditionSucceeded, Reason: pipelinev1beta1.TaskRunReasonFailed.String()})
			failedStep := pipelinev1beta1.StepState{}

			errorReasonValue := "PullBaseImageFailed"
			errorMessageValue := "Failed to pull the base image."
			errorReasonKey := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorReason)
			errorMessageKey := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorMessage)

			errorReason := pipelinev1beta1.PipelineResourceResult{Key: errorReasonKey, Value: errorReasonValue}
			errorMessage := pipelinev1beta1.PipelineResourceResult{Key: errorMessageKey, Value: errorMessageValue}
			unrelated := pipelinev1beta1.PipelineResourceResult{Key: "unrelated-resource-key", Value: "Unrelated resource value"}

			message, _ := json.Marshal([]pipelinev1beta1.PipelineResourceResult{errorReason, errorMessage, unrelated})

			failedStep.Terminated = &corev1.ContainerStateTerminated{Message: string(message), ExitCode: 1}
			followUpStep := pipelinev1beta1.StepState{}

			redTaskRun.Status.Steps = append(redTaskRun.Status.Steps, failedStep, followUpStep)
			redBuild := buildv1alpha1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(ctx, client, &redBuild, &redTaskRun)

			Expect(redBuild.Status.FailureDetails.Message).To(Equal(errorMessageValue))
			Expect(redBuild.Status.FailureDetails.Reason).To(Equal(errorReasonValue))
		})

		It("does not surface unrelated Tekton resources if the TaskRun fails", func() {
			redTaskRun := pipelinev1beta1.TaskRun{}
			redTaskRun.Status.Conditions = append(redTaskRun.Status.Conditions,
				apis.Condition{Type: apis.ConditionSucceeded, Reason: pipelinev1beta1.TaskRunReasonFailed.String()})
			failedStep := pipelinev1beta1.StepState{}

			unrelated := pipelinev1beta1.PipelineResourceResult{Key: "unrelated", Value: "unrelated"}

			message, _ := json.Marshal([]pipelinev1beta1.PipelineResourceResult{unrelated})

			failedStep.Terminated = &corev1.ContainerStateTerminated{Message: string(message)}

			redTaskRun.Status.Steps = append(redTaskRun.Status.Steps, failedStep)
			redBuild := buildv1alpha1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(ctx, client, &redBuild, &redTaskRun)

			Expect(redBuild.Status.FailureDetails.Reason).To(BeEmpty())
			Expect(redBuild.Status.FailureDetails.Message).To(BeEmpty())
		})

		It("does not surface error results if the container terminated without failure", func() {
			greenTaskRun := pipelinev1beta1.TaskRun{}
			greenTaskRun.Status.Conditions = append(greenTaskRun.Status.Conditions,
				apis.Condition{Type: apis.ConditionSucceeded, Reason: pipelinev1beta1.TaskRunReasonSuccessful.String()})
			failedStep := pipelinev1beta1.StepState{}

			errorReasonValue := "PullBaseImageFailed"
			errorMessageValue := "Failed to pull the base image."
			errorReasonKey := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorReason)
			errorMessageKey := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorMessage)

			errorReason := pipelinev1beta1.PipelineResourceResult{Key: errorReasonKey, Value: errorReasonValue}
			errorMessage := pipelinev1beta1.PipelineResourceResult{Key: errorMessageKey, Value: errorMessageValue}
			message, _ := json.Marshal([]pipelinev1beta1.PipelineResourceResult{errorReason, errorMessage})

			failedStep.Terminated = &corev1.ContainerStateTerminated{Message: string(message)}

			greenTaskRun.Status.Steps = append(greenTaskRun.Status.Steps, failedStep)
			greenBuildRun := buildv1alpha1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(ctx, client, &greenBuildRun, &greenTaskRun)

			Expect(greenBuildRun.Status.FailureDetails).To(BeNil())
		})

		It("should not surface errors for a successful TaskRun", func() {
			greenTaskRun := pipelinev1beta1.TaskRun{}
			greenTaskRun.Status.Conditions = append(greenTaskRun.Status.Conditions, apis.Condition{Type: apis.ConditionSucceeded})
			greenBuildRun := buildv1alpha1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(ctx, client, &greenBuildRun, &greenTaskRun)

			Expect(greenBuildRun.Status.FailureDetails).To(BeNil())
		})

		It("should not surface errors if the TaskRun does not have a Succeeded condition", func() {
			unfinishedTaskRun := pipelinev1beta1.TaskRun{}
			unfinishedTaskRun.Status.Conditions = append(unfinishedTaskRun.Status.Conditions, apis.Condition{Type: apis.ConditionReady})
			unfinishedBuildRun := buildv1alpha1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(ctx, client, &unfinishedBuildRun, &unfinishedTaskRun)
			Expect(unfinishedBuildRun.Status.FailureDetails).To(BeNil())
		})

		It("should not surface errors if the TaskRun is in progress", func() {
			unknownTaskRun := pipelinev1beta1.TaskRun{}
			unknownTaskRun.Status.Conditions = append(unknownTaskRun.Status.Conditions, apis.Condition{Type: apis.ConditionSucceeded, Reason: "random"})
			unknownBuildRun := buildv1alpha1.BuildRun{}

			UpdateBuildRunUsingTaskFailures(ctx, client, &unknownBuildRun, &unknownTaskRun)
			Expect(unknownBuildRun.Status.FailureDetails).To(BeNil())
		})
	})
})

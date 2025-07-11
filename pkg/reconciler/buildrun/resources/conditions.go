// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"fmt"
	"time"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"knative.dev/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/ctxlog"
)

// Common condition strings for reason, kind, etc.
const (
	ConditionUnknownStrategyKind                     string = "UnknownStrategyKind"
	ClusterBuildStrategyNotFound                     string = "ClusterBuildStrategyNotFound"
	BuildStrategyNotFound                            string = "BuildStrategyNotFound"
	ConditionSetOwnerReferenceFailed                 string = "SetOwnerReferenceFailed"
	ConditionFailed                                  string = "Failed"
	ConditionTaskRunIsMissing                        string = "TaskRunIsMissing"
	ConditionTaskRunGenerationFailed                 string = "TaskRunGenerationFailed"
	ConditionServiceAccountNotFound                  string = "ServiceAccountNotFound"
	ConditionBuildRegistrationFailed                 string = "BuildRegistrationFailed"
	ConditionBuildNotFound                           string = "BuildNotFound"
	ConditionMissingParameterValues                  string = "MissingParameterValues"
	ConditionRestrictedParametersInUse               string = "RestrictedParametersInUse"
	ConditionUndefinedParameter                      string = "UndefinedParameter"
	ConditionWrongParameterValueType                 string = "WrongParameterValueType"
	ConditionInconsistentParameterValues             string = "InconsistentParameterValues"
	ConditionEmptyArrayItemParameterValues           string = "EmptyArrayItemParameterValues"
	ConditionIncompleteConfigMapValueParameterValues string = "IncompleteConfigMapValueParameterValues"
	ConditionIncompleteSecretValueParameterValues    string = "IncompleteSecretValueParameterValues"
	BuildRunNameInvalid                              string = "BuildRunNameInvalid"
	BuildRunNoRefOrSpec                              string = "BuildRunNoRefOrSpec"
	BuildRunAmbiguousBuild                           string = "BuildRunAmbiguousBuild"
	BuildRunBuildFieldOverrideForbidden              string = "BuildRunBuildFieldOverrideForbidden"
)

// UpdateBuildRunUsingTaskRunCondition updates the BuildRun Succeeded Condition
func UpdateBuildRunUsingTaskRunCondition(ctx context.Context, client client.Client, buildRun *buildv1beta1.BuildRun, taskRun *pipelineapi.TaskRun, trCondition *apis.Condition) error {
	var reason, message string = trCondition.Reason, trCondition.Message
	status := trCondition.Status

	switch pipelineapi.TaskRunReason(reason) {
	case pipelineapi.TaskRunReasonStarted:
		fallthrough
	case pipelineapi.TaskRunReasonRunning:
		if buildRun.IsCanceled() {
			status = corev1.ConditionUnknown // in practice the taskrun status is already unknown in this case, but we are making sure here
			reason = buildv1beta1.BuildRunStateCancel
			message = "The user requested the BuildRun to be canceled.  This BuildRun controller has requested the TaskRun be canceled.  That request has not been process by Tekton's TaskRun controller yet."
		}
	case pipelineapi.TaskRunReasonCancelled:
		if buildRun.IsCanceled() {
			status = corev1.ConditionFalse // in practice the taskrun status is already false in this case, bue we are making sure here
			reason = buildv1beta1.BuildRunStateCancel
			message = "The BuildRun and underlying TaskRun were canceled successfully."
		}

	case pipelineapi.TaskRunReasonTimedOut:
		reason = "BuildRunTimeout"
		var timeout time.Duration
		if taskRun.Spec.Timeout == nil {
			// if the TaskRun does not have a timeout set, we cannot use it to determine the BuildRun timeout
			timeout = time.Since(taskRun.CreationTimestamp.Time)
		} else {
			timeout = taskRun.Spec.Timeout.Duration
		}
		message = fmt.Sprintf("BuildRun %s failed to finish within %s",
			buildRun.Name,
			timeout,
		)

	case pipelineapi.TaskRunReasonSuccessful:
		if buildRun.IsCanceled() {
			message = "The TaskRun completed before the request to cancel the TaskRun could be processed."
		}

	case pipelineapi.TaskRunReasonFailed:
		if taskRun.Status.CompletionTime != nil {
			pod, failedContainer, failedContainerStatus, err := extractFailedPodAndContainer(ctx, client, taskRun)
			if err != nil {
				// when trying to customize the Condition Message field, ensure the Message cover the case
				// when a Pod is deleted.
				// Note: this is an edge case, but not doing this prevent a BuildRun from being marked as Failed
				// while the TaskRun is already with a Failed Reason in it´s condition.
				if apierrors.IsNotFound(err) {
					message = fmt.Sprintf("buildrun failed, pod %s/%s not found", taskRun.Namespace, taskRun.Status.PodName)
					break
				}
				return err
			}

			//nolint:staticcheck // SA1019 we want to give users some time to adopt to failureDetails
			buildRun.Status.FailureDetails = &buildv1beta1.FailureDetails{
				Location: &buildv1beta1.Location{
					Pod: pod.Name,
				},
			}

			if pod.Status.Reason == "Evicted" {
				message = pod.Status.Message
				reason = buildv1beta1.BuildRunStatePodEvicted
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
						reason = buildv1beta1.BuildRunStateStepOutOfMemory
						message = fmt.Sprintf("buildrun step %s failed due to out-of-memory, for detailed information: kubectl --namespace %s logs %s --container=%s",
							failedContainer.Name,
							pod.Namespace,
							pod.Name,
							failedContainer.Name,
						)
					} else if failedContainer.Name == "step-image-processing" && failedContainerStatus.State.Terminated.ExitCode == 22 {
						reason = buildv1beta1.BuildRunStateVulnerabilitiesFound
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

	buildRun.Status.SetCondition(&buildv1beta1.Condition{
		LastTransitionTime: metav1.Now(),
		Type:               buildv1beta1.Succeeded,
		Status:             status,
		Reason:             reason,
		Message:            message,
	})

	return nil
}

// UpdateConditionWithFalseStatus sets the Succeeded condition fields and mark
// the condition as Status False. It also updates the object in the cluster by
// calling client Status Update
func UpdateConditionWithFalseStatus(ctx context.Context, client client.Client, buildRun *buildv1beta1.BuildRun, errorMessage string, reason string) error {
	now := metav1.Now()
	buildRun.Status.CompletionTime = &now
	buildRun.Status.SetCondition(&buildv1beta1.Condition{
		LastTransitionTime: now,
		Type:               buildv1beta1.Succeeded,
		Status:             corev1.ConditionFalse,
		Reason:             reason,
		Message:            errorMessage,
	})
	ctxlog.Debug(ctx, "updating buildRun status", namespace, buildRun.Namespace, name, buildRun.Name, "reason", reason)
	if err := client.Status().Update(ctx, buildRun); err != nil {
		return &ClientStatusUpdateError{err}
	}

	return nil
}

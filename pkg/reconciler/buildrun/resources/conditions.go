// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
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
func UpdateBuildRunUsingTaskRunCondition(ctx context.Context, client client.Client, buildRun *buildv1alpha1.BuildRun, taskRun *v1beta1.TaskRun, trCondition *apis.Condition) error {
	var reason, message string = trCondition.Reason, trCondition.Message
	status := trCondition.Status

	switch v1beta1.TaskRunReason(reason) {
	case v1beta1.TaskRunReasonStarted:
		fallthrough
	case v1beta1.TaskRunReasonRunning:
		if buildRun.IsCanceled() {
			status = corev1.ConditionUnknown // in practice the taskrun status is already unknown in this case, but we are making sure here
			reason = buildv1alpha1.BuildRunStateCancel
			message = "The user requested the BuildRun to be canceled.  This BuildRun controller has requested the TaskRun be canceled.  That request has not been process by Tekton's TaskRun controller yet."
		}
	case v1beta1.TaskRunReasonCancelled:
		if buildRun.IsCanceled() {
			status = corev1.ConditionFalse // in practice the taskrun status is already false in this case, bue we are making sure here
			reason = buildv1alpha1.BuildRunStateCancel
			message = "The BuildRun and underlying TaskRun were canceled successfully."
		}

	case v1beta1.TaskRunReasonTimedOut:
		reason = "BuildRunTimeout"
		message = fmt.Sprintf("BuildRun %s failed to finish within %s",
			buildRun.Name,
			taskRun.Spec.Timeout.Duration,
		)

	case v1beta1.TaskRunReasonSuccessful:
		if buildRun.IsCanceled() {
			message = "The TaskRun completed before the request to cancel the TaskRun could be processed."
		}

	case v1beta1.TaskRunReasonFailed:
		if taskRun.Status.CompletionTime != nil {
			pod, failedContainer, err := extractFailedPodAndContainer(ctx, client, taskRun)
			if err != nil {
				// when trying to customize the Condition Message field, ensure the Message cover the case
				// when a Pod is deleted.
				// Note: this is an edge case, but not doing this prevent a BuildRun from being marked as Failed
				// while the TaskRun is already with a Failed Reason in it´s condition.
				if apierrors.IsNotFound(err) {
					message = fmt.Sprintf("buildrun failed, pod %s not found", taskRun.Status.PodName)
					break
				}
				return err
			}

			//nolint:staticcheck // SA1019 we want to give users some time to adopt to failureDetails
			buildRun.Status.FailedAt = &buildv1alpha1.FailedAt{Pod: pod.Name}

			if failedContainer != nil {
				//nolint:staticcheck // SA1019 we want to give users some time to adopt to failureDetails
				buildRun.Status.FailedAt.Container = failedContainer.Name
				message = fmt.Sprintf("buildrun step %s failed in pod %s, for detailed information: kubectl --namespace %s logs %s --container=%s",
					failedContainer.Name,
					pod.Name,
					pod.Namespace,
					pod.Name,
					failedContainer.Name,
				)
			} else if pod.Status.Reason == "Evicted" {
				message = pod.Status.Message
				reason = buildv1alpha1.BuildRunStatePodEvicted
			} else {
				message = fmt.Sprintf("buildrun failed due to an unexpected error in pod %s: for detailed information: kubectl --namespace %s logs %s --all-containers",
					pod.Name,
					pod.Namespace,
					pod.Name,
				)
			}
		}
	}

	buildRun.Status.SetCondition(&buildv1alpha1.Condition{
		LastTransitionTime: metav1.Now(),
		Type:               buildv1alpha1.Succeeded,
		Status:             status,
		Reason:             reason,
		Message:            message,
	})

	return nil
}

// UpdateConditionWithFalseStatus sets the Succeeded condition fields and mark
// the condition as Status False. It also updates the object in the cluster by
// calling client Status Update
func UpdateConditionWithFalseStatus(ctx context.Context, client client.Client, buildRun *buildv1alpha1.BuildRun, errorMessage string, reason string) error {
	now := metav1.Now()
	buildRun.Status.CompletionTime = &now
	buildRun.Status.SetCondition(&buildv1alpha1.Condition{
		LastTransitionTime: now,
		Type:               buildv1alpha1.Succeeded,
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

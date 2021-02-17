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
	"k8s.io/apimachinery/pkg/types"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

// UpdateBuildRunUsingTaskRunCondition updates the BuildRun Succeeded Condition
func UpdateBuildRunUsingTaskRunCondition(ctx context.Context, client client.Client, buildRun *buildv1alpha1.BuildRun, taskRun *v1beta1.TaskRun, trCondition *apis.Condition) error {
	var reason, message string = trCondition.Reason, trCondition.Message

	switch v1beta1.TaskRunReason(reason) {
	case v1beta1.TaskRunReasonTimedOut:
		reason = "BuildRunTimeout"
		message = fmt.Sprintf("BuildRun %s failed to finish within %s",
			buildRun.Name,
			taskRun.Spec.Timeout.Duration,
		)

	case v1beta1.TaskRunReasonFailed:
		if taskRun.Status.CompletionTime != nil {
			var pod corev1.Pod
			if err := client.Get(ctx, types.NamespacedName{Namespace: taskRun.Namespace, Name: taskRun.Status.PodName}, &pod); err != nil {
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

			buildRun.Status.FailedAt = &buildv1alpha1.FailedAt{Pod: pod.Name}

			// Since the container status list is not sorted, as a quick workaround mark all failed containers
			var failures = make(map[string]struct{})
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode != 0 {
					failures[containerStatus.Name] = struct{}{}
				}
			}

			// Find the first container that failed
			var failedContainer *corev1.Container
			for i, container := range pod.Spec.Containers {
				if _, has := failures[container.Name]; has {
					failedContainer = &pod.Spec.Containers[i]
					break
				}
			}

			if failedContainer != nil {
				buildRun.Status.FailedAt.Container = failedContainer.Name
				message = fmt.Sprintf("buildrun step failed in pod %s, for detailed information: kubectl --namespace %s logs %s --container=%s",
					pod.Name,
					pod.Namespace,
					pod.Name,
					failedContainer.Name,
				)
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
		Status:             trCondition.Status,
		Reason:             reason,
		Message:            message,
	})

	return nil
}

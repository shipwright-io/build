// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

func (t *TestBuild) LookupSecret(entity types.NamespacedName) (*corev1.Secret, error) {
	result, err := t.lookupRuntimeObject(func(ctx context.Context) (runtime.Object, error) {
		return t.Clientset.
			CoreV1().
			Secrets(entity.Namespace).
			Get(ctx, entity.Name, metav1.GetOptions{})
	})

	return result.(*corev1.Secret), err
}

func (t *TestBuild) LookupPod(entity types.NamespacedName) (*corev1.Pod, error) {
	result, err := t.lookupRuntimeObject(func(ctx context.Context) (runtime.Object, error) {
		return t.Clientset.
			CoreV1().
			Pods(entity.Namespace).
			Get(ctx, entity.Name, metav1.GetOptions{})
	})

	return result.(*corev1.Pod), err
}

func (t *TestBuild) LookupBuild(entity types.NamespacedName) (*buildv1beta1.Build, error) {
	result, err := t.lookupRuntimeObject(func(ctx context.Context) (runtime.Object, error) {
		return t.BuildClientSet.ShipwrightV1beta1().
			Builds(entity.Namespace).Get(ctx, entity.Name, metav1.GetOptions{})
	})

	return result.(*buildv1beta1.Build), err
}

func (t *TestBuild) LookupBuildRun(entity types.NamespacedName) (*buildv1beta1.BuildRun, error) {
	result, err := t.lookupRuntimeObject(func(ctx context.Context) (runtime.Object, error) {
		return t.BuildClientSet.ShipwrightV1beta1().
			BuildRuns(entity.Namespace).Get(ctx, entity.Name, metav1.GetOptions{})
	})

	return result.(*buildv1beta1.BuildRun), err
}

func (t *TestBuild) LookupTaskRun(entity types.NamespacedName) (*pipelineapi.TaskRun, error) {
	result, err := t.lookupRuntimeObject(func(ctx context.Context) (runtime.Object, error) {
		return t.PipelineClientSet.
			TektonV1().
			TaskRuns(entity.Namespace).
			Get(ctx, entity.Name, metav1.GetOptions{})
	})

	return result.(*pipelineapi.TaskRun), err
}

func (t *TestBuild) LookupTaskRunUsingBuildRun(buildRun *buildv1beta1.BuildRun) (*pipelineapi.TaskRun, error) {
	if buildRun == nil {
		return nil, fmt.Errorf("no BuildRun specified to lookup TaskRun")
	}

	if buildRun.Status.TaskRunName != nil {
		return t.LookupTaskRun(types.NamespacedName{Namespace: buildRun.Namespace, Name: *buildRun.Status.TaskRunName})
	}

	tmp, err := t.lookupRuntimeObject(func(ctx context.Context) (runtime.Object, error) {
		return t.PipelineClientSet.
			TektonV1().
			TaskRuns(buildRun.Namespace).
			List(ctx, metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(
					map[string]string{
						buildv1beta1.LabelBuildRun: buildRun.Name,
					}).String(),
			})
	})

	if err != nil {
		return nil, err
	}

	var taskRunList = tmp.(*pipelineapi.TaskRunList)
	switch len(taskRunList.Items) {
	case 0:
		return nil, fmt.Errorf("no TaskRun found for BuildRun %s/%s", buildRun.Namespace, buildRun.Name)

	case 1:
		return &taskRunList.Items[0], nil

	default:
		return nil, fmt.Errorf("multiple TaskRuns found for BuildRun %s/%s, which should not have happened", buildRun.Namespace, buildRun.Name)
	}
}

func (t *TestBuild) LookupServiceAccount(entity types.NamespacedName) (*corev1.ServiceAccount, error) {
	result, err := t.lookupRuntimeObject(func(ctx context.Context) (runtime.Object, error) {
		return t.Clientset.
			CoreV1().
			ServiceAccounts(entity.Namespace).
			Get(ctx, entity.Name, metav1.GetOptions{})
	})

	return result.(*corev1.ServiceAccount), err
}

func (t *TestBuild) lookupRuntimeObject(f func(ctx context.Context) (runtime.Object, error)) (result runtime.Object, err error) {
	err = wait.PollUntilContextTimeout(t.Context, 4*time.Second, 60*time.Second, true, func(ctx context.Context) (bool, error) {
		result, err = f(ctx)
		if err != nil {
			// check if we have an error that we want to retry
			if isRetryableError(err) {
				return false, nil
			}

			return true, err
		}

		// successful call
		return true, nil
	})

	return
}

func isRetryableError(err error) bool {
	return apierrors.IsServerTimeout(err) ||
		apierrors.IsTimeout(err) ||
		apierrors.IsTooManyRequests(err) ||
		apierrors.IsInternalError(err) ||
		err.Error() == "etcdserver: request timed out" ||
		err.Error() == "rpc error: code = Unavailable desc = transport is closing" ||
		strings.Contains(err.Error(), "net/http: request canceled while waiting for connection")
}

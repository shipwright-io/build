// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"strings"
	"time"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

func (t *TestBuild) LookupSecret(entity types.NamespacedName) (*corev1.Secret, error) {
	result, err := lookupRuntimeObject(func() (runtime.Object, error) {
		return t.Clientset.
			CoreV1().
			Secrets(entity.Namespace).
			Get(t.Context, entity.Name, metav1.GetOptions{})
	})

	return result.(*corev1.Secret), err
}

func (t *TestBuild) LookupPod(entity types.NamespacedName) (*corev1.Pod, error) {
	result, err := lookupRuntimeObject(func() (runtime.Object, error) {
		return t.Clientset.
			CoreV1().
			Pods(entity.Namespace).
			Get(t.Context, entity.Name, metav1.GetOptions{})
	})

	return result.(*corev1.Pod), err
}

func (t *TestBuild) LookupBuild(entity types.NamespacedName) (*buildv1alpha1.Build, error) {
	result, err := lookupRuntimeObject(func() (runtime.Object, error) {
		return t.BuildClientSet.ShipwrightV1alpha1().
			Builds(entity.Namespace).Get(t.Context, entity.Name, metav1.GetOptions{})
	})

	return result.(*buildv1alpha1.Build), err
}

func (t *TestBuild) LookupBuildRun(entity types.NamespacedName) (*buildv1alpha1.BuildRun, error) {
	result, err := lookupRuntimeObject(func() (runtime.Object, error) {
		return t.BuildClientSet.ShipwrightV1alpha1().
			BuildRuns(entity.Namespace).Get(t.Context, entity.Name, metav1.GetOptions{})
	})

	return result.(*buildv1alpha1.BuildRun), err
}

func (t *TestBuild) LookupTaskRun(entity types.NamespacedName) (*pipelinev1beta1.TaskRun, error) {
	result, err := lookupRuntimeObject(func() (runtime.Object, error) {
		return t.PipelineClientSet.
			TektonV1beta1().
			TaskRuns(entity.Namespace).
			Get(t.Context, entity.Name, metav1.GetOptions{})
	})

	return result.(*pipelinev1beta1.TaskRun), err
}

func (t *TestBuild) LookupTaskRunUsingBuildRun(buildRun *buildv1alpha1.BuildRun) (*pipelinev1beta1.TaskRun, error) {
	if buildRun == nil {
		return nil, fmt.Errorf("no BuildRun specified to lookup TaskRun")
	}

	if buildRun.Status.LatestTaskRunRef != nil {
		return t.LookupTaskRun(types.NamespacedName{Namespace: buildRun.Namespace, Name: *buildRun.Status.LatestTaskRunRef})
	}

	tmp, err := lookupRuntimeObject(func() (runtime.Object, error) {
		return t.PipelineClientSet.
			TektonV1beta1().
			TaskRuns(buildRun.Namespace).
			List(t.Context, metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(
					map[string]string{
						buildv1alpha1.LabelBuildRun: buildRun.Name,
					}).String(),
			})
	})

	if err != nil {
		return nil, err
	}

	var taskRunList = tmp.(*pipelinev1beta1.TaskRunList)
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
	result, err := lookupRuntimeObject(func() (runtime.Object, error) {
		return t.Clientset.
			CoreV1().
			ServiceAccounts(entity.Namespace).
			Get(t.Context, entity.Name, metav1.GetOptions{})
	})

	return result.(*corev1.ServiceAccount), err
}

func lookupRuntimeObject(f func() (runtime.Object, error)) (result runtime.Object, err error) {
	err = wait.PollImmediate(4*time.Second, 60*time.Second, func() (bool, error) {
		result, err = f()
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

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/test/integration/utils"
)

func lookupSecret(build *utils.TestBuild, entity types.NamespacedName) (*corev1.Secret, error) {
	result, err := lookupRuntimeObject(testBuild, func() (runtime.Object, error) {
		return testBuild.
			Clientset.
			CoreV1().
			Secrets(entity.Namespace).
			Get(testBuild.Context, entity.Name, metav1.GetOptions{})
	})

	if err != nil {
		return nil, err
	}

	return result.(*corev1.Secret), nil
}

func lookupPod(testBuild *utils.TestBuild, entity types.NamespacedName) (*corev1.Pod, error) {
	result, err := lookupRuntimeObject(testBuild, func() (runtime.Object, error) {
		return testBuild.
			Clientset.
			CoreV1().
			Pods(entity.Namespace).
			Get(testBuild.Context, entity.Name, metav1.GetOptions{})
	})

	if err != nil {
		return nil, err
	}

	return result.(*corev1.Pod), nil
}

func lookupBuild(testBuild *utils.TestBuild, entity types.NamespacedName) (*buildv1alpha1.Build, error) {
	result, err := lookupRuntimeObject(testBuild, func() (runtime.Object, error) {
		return testBuild.GetBuild(entity.Name)
	})

	if err != nil {
		return nil, err
	}

	return result.(*buildv1alpha1.Build), nil
}

func lookupBuildRun(testBuild *utils.TestBuild, entity types.NamespacedName) (*buildv1alpha1.BuildRun, error) {
	result, err := lookupRuntimeObject(testBuild, func() (runtime.Object, error) {
		return testBuild.GetBR(entity.Name)
	})

	if err != nil {
		return nil, err
	}

	return result.(*buildv1alpha1.BuildRun), nil
}

func lookupTaskRun(testBuild *utils.TestBuild, entity types.NamespacedName) (*pipelinev1beta1.TaskRun, error) {
	result, err := lookupRuntimeObject(testBuild, func() (runtime.Object, error) {
		return testBuild.
			PipelineClientSet.
			TektonV1beta1().
			TaskRuns(entity.Namespace).
			Get(testBuild.Context, entity.Name, metav1.GetOptions{})
	})

	if err != nil {
		return nil, err
	}

	return result.(*pipelinev1beta1.TaskRun), nil
}

func lookupServiceAccount(testBuild *utils.TestBuild, entity types.NamespacedName) (*corev1.ServiceAccount, error) {
	result, err := lookupRuntimeObject(testBuild, func() (runtime.Object, error) {
		return testBuild.
			Clientset.
			CoreV1().
			ServiceAccounts(entity.Namespace).
			Get(testBuild.Context, entity.Name, metav1.GetOptions{})
	})

	if err != nil {
		return nil, err
	}

	return result.(*corev1.ServiceAccount), nil
}

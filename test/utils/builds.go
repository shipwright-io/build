// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

// This class is intended to host all CRUD calls for testing Build CRDs resources

// CreateBuild generates a Build on the current test namespace
func (t *TestBuild) CreateBuild(build *v1alpha1.Build) error {
	bInterface := t.BuildClientSet.BuildV1alpha1().Builds(t.Namespace)

	_, err := bInterface.Create(context.TODO(), build, metav1.CreateOptions{})
	return err
}

// DeleteBuild deletes a Build on the desired namespace
func (t *TestBuild) DeleteBuild(name string) error {
	bInterface := t.BuildClientSet.BuildV1alpha1().Builds(t.Namespace)

	err := bInterface.Delete(context.TODO(), name, metav1.DeleteOptions{})

	return err
}

// GetBuild returns a Build based on name
func (t *TestBuild) GetBuild(name string) (*v1alpha1.Build, error) {
	return t.BuildClientSet.
		BuildV1alpha1().
		Builds(t.Namespace).
		Get(context.TODO(), name, metav1.GetOptions{})
}

// PatchBuild patches an existing Build using the merge patch type
func (t *TestBuild) PatchBuild(buildName string, data []byte) (*v1alpha1.Build, error) {
	return t.PatchBuildWithPatchType(buildName, data, types.MergePatchType)
}

// PatchBuildWithPatchType patches an existing Build and allows specifying the patch type
func (t *TestBuild) PatchBuildWithPatchType(buildName string, data []byte, pt types.PatchType) (*v1alpha1.Build, error) {
	bInterface := t.BuildClientSet.BuildV1alpha1().Builds(t.Namespace)
	b, err := bInterface.Patch(context.TODO(), buildName, pt, data, metav1.PatchOptions{})
	if err != nil {
		return nil, err
	}
	return b, nil
}

// GetBuildTillValidation polls until a Build gets a validation and updates
// it´s registered field. If timeout is reached or an error is found, it will
// return with an error
func (t *TestBuild) GetBuildTillValidation(name string) (*v1alpha1.Build, error) {

	var (
		pollBuildTillRegistration = func() (bool, error) {

			bInterface := t.BuildClientSet.BuildV1alpha1().Builds(t.Namespace)

			buildRun, err := bInterface.Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}
			// TODO: we might improve the conditional here
			if buildRun.Status.Registered != "" {
				return true, nil
			}

			return false, nil
		}
	)

	brInterface := t.BuildClientSet.BuildV1alpha1().Builds(t.Namespace)

	if err := wait.PollImmediate(t.Interval, t.TimeOut, pollBuildTillRegistration); err != nil {
		return nil, err
	}

	return brInterface.Get(context.TODO(), name, metav1.GetOptions{})
}

// GetBuildTillRegistration polls until a Build gets a desired validation and updates
// it´s registered field. If timeout is reached or an error is found, it will
// return with an error
func (t *TestBuild) GetBuildTillRegistration(name string, condition corev1.ConditionStatus) (*v1alpha1.Build, error) {

	var (
		pollBuildTillRegistration = func() (bool, error) {

			bInterface := t.BuildClientSet.BuildV1alpha1().Builds(t.Namespace)

			buildRun, err := bInterface.Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}
			// TODO: we might improve the conditional here
			if buildRun.Status.Registered == condition {
				return true, nil
			}

			return false, nil
		}
	)

	brInterface := t.BuildClientSet.BuildV1alpha1().Builds(t.Namespace)

	if err := wait.PollImmediate(t.Interval, t.TimeOut, pollBuildTillRegistration); err != nil {
		return nil, err
	}

	return brInterface.Get(context.TODO(), name, metav1.GetOptions{})
}

// GetBuildTillMessageContainsSubstring polls until a Build message contains the desired
// substring value and updates it´s registered field. If timeout is reached or an error is found,
// it will return with an error
func (t *TestBuild) GetBuildTillMessageContainsSubstring(name string, partOfMessage string) (*v1alpha1.Build, error) {

	var (
		pollBuildTillMessageContainsSubString = func() (bool, error) {

			bInterface := t.BuildClientSet.BuildV1alpha1().Builds(t.Namespace)

			buildRun, err := bInterface.Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}

			if strings.Contains(buildRun.Status.Message, partOfMessage) {
				return true, nil
			}

			return false, nil
		}
	)

	brInterface := t.BuildClientSet.BuildV1alpha1().Builds(t.Namespace)

	if err := wait.PollImmediate(t.Interval, t.TimeOut, pollBuildTillMessageContainsSubString); err != nil {
		return nil, err
	}

	return brInterface.Get(context.TODO(), name, metav1.GetOptions{})
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

// This class is intended to host all CRUD calls for testing Build CRDs resources

// CreateBuild generates a Build on the current test namespace
func (t *TestBuild) CreateBuild(build *v1alpha1.Build) error {
	bInterface := t.BuildClientSet.BuildV1alpha1().Builds(t.Namespace)

	_, err := bInterface.Create(build)
	return err
}

// DeleteBuild deletes a Build on the desired namespace
func (t *TestBuild) DeleteBuild(name string) error {
	bInterface := t.BuildClientSet.BuildV1alpha1().Builds(t.Namespace)

	err := bInterface.Delete(name, &metav1.DeleteOptions{})

	return err
}

// GetBuild returns a Build based on name
func (t *TestBuild) GetBuild(name string) (*v1alpha1.Build, error) {
	return t.BuildClientSet.
		BuildV1alpha1().
		Builds(t.Namespace).
		Get(name, metav1.GetOptions{})
}

// PatchBuild patches an existing Build using the merge patch type
func (t *TestBuild) PatchBuild(buildName string, data []byte) (*v1alpha1.Build, error) {
	return t.PatchBuildWithPatchType(buildName, data, types.MergePatchType)
}

// PatchBuildWithPatchType patches an existing Build and allows specifying the patch type
func (t *TestBuild) PatchBuildWithPatchType(buildName string, data []byte, pt types.PatchType) (*v1alpha1.Build, error) {
	bInterface := t.BuildClientSet.BuildV1alpha1().Builds(t.Namespace)
	b, err := bInterface.Patch(buildName, pt, data)
	if err != nil {
		return nil, err
	}
	return b, nil
}

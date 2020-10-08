// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	bInterface := t.BuildClientSet.BuildV1alpha1().Builds(t.Namespace)

	build, err := bInterface.Get(name, metav1.GetOptions{})
	if err != nil {
		return build, err
	}
	return nil, nil
}

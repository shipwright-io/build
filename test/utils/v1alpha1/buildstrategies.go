// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

// This class is intended to host all CRUD calls for testing BuildStrategy CRDs resources

// CreateBuildStrategy generates a BuildStrategy on the current test namespace
func (t *TestBuild) CreateBuildStrategy(bs *v1alpha1.BuildStrategy) error {
	bsInterface := t.BuildClientSet.ShipwrightV1alpha1().BuildStrategies(t.Namespace)

	_, err := bsInterface.Create(t.Context, bs, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// DeleteBuildStrategy deletes a BuildStrategy on the current test namespace
func (t *TestBuild) DeleteBuildStrategy(name string) error {
	bsInterface := t.BuildClientSet.ShipwrightV1alpha1().BuildStrategies(t.Namespace)

	return bsInterface.Delete(t.Context, name, metav1.DeleteOptions{})
}

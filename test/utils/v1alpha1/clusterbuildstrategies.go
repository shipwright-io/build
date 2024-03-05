// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

// This class is intended to host all CRUD calls for testing ClusterBuildStrategy CRDs resources

// CreateClusterBuildStrategy generates a ClusterBuildStrategy on the current test namespace
func (t *TestBuild) CreateClusterBuildStrategy(cbs *v1alpha1.ClusterBuildStrategy) error {
	cbsInterface := t.BuildClientSet.ShipwrightV1alpha1().ClusterBuildStrategies()

	_, err := cbsInterface.Create(t.Context, cbs, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// DeleteClusterBuildStrategy deletes a ClusterBuildStrategy on the desired namespace
func (t *TestBuild) DeleteClusterBuildStrategy(name string) error {
	cbsInterface := t.BuildClientSet.ShipwrightV1alpha1().ClusterBuildStrategies()

	err := cbsInterface.Delete(t.Context, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

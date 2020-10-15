// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This class is intended to host all CRUD calls for testing ClusterBuildStrategy CRDs resources

// CreateClusterBuildStrategy generates a ClusterBuildStrategy on the current test namespace
func (t *TestBuild) CreateClusterBuildStrategy(ctx context.Context, cbs *v1alpha1.ClusterBuildStrategy) error {
	cbsInterface := t.BuildClientSet.BuildV1alpha1().ClusterBuildStrategies()

	_, err := cbsInterface.Create(ctx, cbs, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// DeleteClusterBuildStrategy deletes a ClusterBuildStrategy on the desired namespace
func (t *TestBuild) DeleteClusterBuildStrategy(ctx context.Context, name string) error {
	cbsInterface := t.BuildClientSet.BuildV1alpha1().ClusterBuildStrategies()

	err := cbsInterface.Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

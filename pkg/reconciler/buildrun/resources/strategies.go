// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/shipwright-io/build/pkg/ctxlog"
	"k8s.io/apimachinery/pkg/types"
)

// RetrieveBuildStrategy returns a namespace scoped strategy
func RetrieveBuildStrategy(ctx context.Context, client client.Client, build *buildv1alpha1.Build) (*buildv1alpha1.BuildStrategy, error) {
	buildStrategyInstance := &buildv1alpha1.BuildStrategy{}

	ctxlog.Debug(ctx, "retrieving BuildStrategy", namespace, build.Namespace, name, build.Name)
	if err := client.Get(ctx, types.NamespacedName{Name: build.Spec.StrategyRef.Name, Namespace: build.Namespace}, buildStrategyInstance); err != nil {
		return nil, err
	}
	return buildStrategyInstance, nil
}

// RetrieveClusterBuildStrategy returns a cluster scoped strategy
func RetrieveClusterBuildStrategy(ctx context.Context, client client.Client, build *buildv1alpha1.Build) (*buildv1alpha1.ClusterBuildStrategy, error) {
	clusterBuildStrategyInstance := &buildv1alpha1.ClusterBuildStrategy{}

	ctxlog.Debug(ctx, "retrieving ClusterBuildStrategy", namespace, build.Namespace, name, build.Name)
	if err := client.Get(ctx, types.NamespacedName{Name: build.Spec.StrategyRef.Name}, clusterBuildStrategyInstance); err != nil {
		return nil, err
	}
	return clusterBuildStrategyInstance, nil
}

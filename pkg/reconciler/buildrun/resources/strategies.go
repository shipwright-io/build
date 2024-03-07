// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/shipwright-io/build/pkg/ctxlog"
	"k8s.io/apimachinery/pkg/types"
)

// RetrieveBuildStrategy returns a namespace scoped strategy
func RetrieveBuildStrategy(ctx context.Context, client client.Client, build *buildapi.Build) (*buildapi.BuildStrategy, error) {
	buildStrategyInstance := &buildapi.BuildStrategy{}

	ctxlog.Debug(ctx, "retrieving BuildStrategy", namespace, build.Namespace, name, build.Name)

	// Note: When returning the client.Get call, the buildStrategyInstance gets populated and properly returned as the first argument
	return buildStrategyInstance, client.Get(ctx, types.NamespacedName{Name: build.Spec.Strategy.Name, Namespace: build.Namespace}, buildStrategyInstance)
}

// RetrieveClusterBuildStrategy returns a cluster scoped strategy
func RetrieveClusterBuildStrategy(ctx context.Context, client client.Client, build *buildapi.Build) (*buildapi.ClusterBuildStrategy, error) {
	clusterBuildStrategyInstance := &buildapi.ClusterBuildStrategy{}

	ctxlog.Debug(ctx, "retrieving ClusterBuildStrategy", namespace, build.Namespace, name, build.Name)

	// Note: When returning the client.Get call, the clusterBuildStrategyInstance gets populated and properly returned as the first argument
	return clusterBuildStrategyInstance, client.Get(ctx, types.NamespacedName{Name: build.Spec.Strategy.Name}, clusterBuildStrategyInstance)
}

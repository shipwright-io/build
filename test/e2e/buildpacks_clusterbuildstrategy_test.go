package e2e

import (
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
)

// buildpacks-v3 Test data by using ClusterBuildStrategy setup
func buildpackBuildTestDataFromClusterBuildStrategy(ns string, identifier string) (*operator.Build, *operator.ClusterBuildStrategy, error) {
	return buildTestDataFromClusterBuildStrategy(ns, identifier,
		"samples/buildstrategy/buildpacks-v3/clusterbuildstrategy_buildpacks-v3_cr.yaml",
		"samples/build/build_buildpacks-v3_clusterbuildstrategy_cr.yaml")
}

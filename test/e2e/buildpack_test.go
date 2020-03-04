package e2e

import (
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
)

// buildpacks-v3 Test data setup
func buildpackBuildTestData(ns string, identifier string) (*operator.Build, *operator.BuildStrategy, error) {
	return buildTestData(ns, identifier,
		"samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml",
		"samples/build/build_buildpacks-v3_cr.yaml")
}

package e2e

import (
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
)

// buildahBuild Test data setup
func buildahBuildTestData(ns string, identifier string) (*operator.Build, *operator.BuildStrategy, error) {

	return buildTestData(ns, identifier,
		"samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml",
		"samples/build/build_buildah_cr.yaml")
}

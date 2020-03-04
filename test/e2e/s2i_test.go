package e2e

import (
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
)

// s2iBuildTestData Test data setup
func s2iBuildTestData(ns string, identifier string) (*operator.Build, *operator.BuildStrategy, error) {
	return buildTestData(ns, identifier,
		"samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml",
		"samples/build/build_source-to-image_cr.yaml")
}

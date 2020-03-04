package e2e

import (
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
)

// kanikoBuildTestData Test data setup
func kanikoBuildTestData(ns string, identifier string) (*operator.Build, *operator.BuildStrategy, error) {
	return buildTestData(ns, identifier,
		"samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml",
		"samples/build/build_kaniko_cr.yaml")
}

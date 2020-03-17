package e2e

import (
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
)

// s2iBuildTestData Test data setup
func s2iBuildTestData(ns string, identifier string) (*operator.ClusterBuildStrategy, *operator.Build, *operator.BuildRun, error) {
	buildstrategy, err := clusterBuildStrategyTestData("samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml")
	if err != nil {
		return nil, nil, nil, err
	}

	build, err := buildTestData(ns, identifier, "samples/build/build_source-to-image_cr.yaml")
	if err != nil {
		return nil, nil, nil, err
	}

	buildRun, err := buildRunTestData(ns, identifier, "samples/buildrun/buildrun_source-to-image_cr.yaml")
	if err != nil {
		return nil, nil, nil, err
	}

	return buildstrategy, build, buildRun, nil
}

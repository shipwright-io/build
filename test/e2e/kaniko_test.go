package e2e

import (
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
)

// kanikoBuildTestData Test data setup
func kanikoBuildTestData(ns string, identifier string) (*operator.ClusterBuildStrategy, *operator.Build, *operator.BuildRun, error) {
	buildstrategy, err := clusterBuildStrategyTestData("samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml")
	if err != nil {
		return nil, nil, nil, err
	}

	build, err := buildTestData(ns, identifier, "samples/build/build_kaniko_cr.yaml")
	if err != nil {
		return nil, nil, nil, err
	}

	buildRun, err := buildRunTestData(ns, identifier, "samples/buildrun/buildrun_kaniko_cr.yaml")
	if err != nil {
		return nil, nil, nil, err
	}

	return buildstrategy, build, buildRun, nil
}

package e2e

import (
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
)

// buildahBuild Test data setup
func buildahBuildTestData(ns string, identifier string) (*operator.ClusterBuildStrategy, *operator.Build, *operator.BuildRun, error) {
	buildstrategy, err := clusterBuildStrategyTestData("samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml")
	if err != nil {
		return nil, nil, nil, err
	}

	build, err := buildTestData(ns, identifier, "samples/build/build_buildah_cr.yaml")
	if err != nil {
		return nil, nil, nil, err
	}

	buildRun, err := buildRunTestData(ns, identifier, "samples/buildrun/buildrun_buildah_cr.yaml")
	if err != nil {
		return nil, nil, nil, err
	}

	return buildstrategy, build, buildRun, nil
}

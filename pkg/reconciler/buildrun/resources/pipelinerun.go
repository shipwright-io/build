// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

// GeneratePipelineRun creates a PipelineRun with separate tasks for each build phase.
func GeneratePipelineRun(
	cfg *config.Config,
	build *buildv1beta1.Build,
	buildRun *buildv1beta1.BuildRun,
	serviceAccountName string,
	strategy buildv1beta1.BuilderStrategy,
) (*pipelineapi.PipelineRun, error) {
	generator := NewPipelineRunGenerator(cfg, build, buildRun, serviceAccountName, strategy)

	executor, err := GenerateBuildRunExecutor(build, buildRun, strategy, generator)
	if err != nil {
		return nil, err
	}

	pipelineRun, ok := executor.(*pipelineapi.PipelineRun)
	if !ok {
		return nil, fmt.Errorf("expected PipelineRun but got %T", executor)
	}

	return pipelineRun, nil
}

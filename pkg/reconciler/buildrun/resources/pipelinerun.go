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

// GeneratePipelineRun creates a Tekton PipelineRun object for a build run.
//
// Execution Model (PipelineRun):
//
//	PipelineRun
//	  ├─ Task 1: source-acquisition
//	  │    └─ Step: git-clone
//	  ├─ Task 2: build-strategy
//	  │    └─ Step: buildah-build
//	  └─ Task 3: output-image
//	       └─ Step: image-push
//
// This function uses the Template Method pattern via GenerateBuildRunExecutor.
// Each phase runs as a separate Task with data transfer via workspace volumes (PVC templates).
// This enables better resource isolation and future multi-arch parallel builds.
func GeneratePipelineRun(
	cfg *config.Config,
	build *buildv1beta1.Build,
	buildRun *buildv1beta1.BuildRun,
	serviceAccountName string,
	strategy buildv1beta1.BuilderStrategy,
) (*pipelineapi.PipelineRun, error) {
	// Create PipelineRun generator
	generator := NewPipelineRunGenerator(cfg, build, buildRun, serviceAccountName, strategy)

	// Use the Template Method to generate the executor
	executor, err := GenerateBuildRunExecutor(cfg, build, buildRun, serviceAccountName, strategy, generator)
	if err != nil {
		return nil, err
	}

	// Type assert to PipelineRun (safe because we know the generator returns a PipelineRun)
	pipelineRun, ok := executor.(*pipelineapi.PipelineRun)
	if !ok {
		return nil, fmt.Errorf("expected PipelineRun but got %T", executor)
	}

	return pipelineRun, nil
}

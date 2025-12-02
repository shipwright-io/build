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

const (
	prefixParamsResultsVolumes = "shp"

	paramOutputImage     = "output-image"
	paramOutputInsecure  = "output-insecure"
	paramSourceRoot      = "source-root"
	paramSourceContext   = "source-context"
	paramOutputDirectory = "output-directory"

	workspaceSource = "source"
)

// GenerateTaskRun creates a Tekton TaskRun to be used for a build run.
//
// Execution Model (TaskRun):
//
//	TaskRun
//	  └─ TaskSpec (inline)
//	       ├─ Step: git-clone         (Phase 1)
//	       ├─ Step: buildah-build     (Phase 2)
//	       └─ Step: image-push        (Phase 3)
//
// This function uses the Template Method pattern via GenerateBuildRunExecutor.
// All phases are combined as sequential steps within a single TaskSpec.
func GenerateTaskRun(
	cfg *config.Config,
	build *buildv1beta1.Build,
	buildRun *buildv1beta1.BuildRun,
	serviceAccountName string,
	strategy buildv1beta1.BuilderStrategy,
) (*pipelineapi.TaskRun, error) {
	// Create TaskRun generator
	generator := NewTaskRunGenerator(cfg, build, buildRun, serviceAccountName, strategy)

	// Use the Template Method to generate the executor
	executor, err := GenerateBuildRunExecutor(cfg, build, buildRun, serviceAccountName, strategy, generator)
	if err != nil {
		return nil, err
	}

	// Type assert to TaskRun (safe because we know the generator returns a TaskRun)
	taskRun, ok := executor.(*pipelineapi.TaskRun)
	if !ok {
		return nil, fmt.Errorf("expected TaskRun but got %T", executor)
	}

	return taskRun, nil
}

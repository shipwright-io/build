// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
)

const (
	prefixParamsResultsVolumes = "shp"

	paramOutputImage     = "output-image"
	paramOutputInsecure  = "output-insecure"
	paramSourceRoot      = "source-root"
	paramSourceContext   = "source-context"
	paramOutputDirectory = "output-directory"
	paramCABundle        = "ca-bundle"

	workspaceSource = "source"
)

// GenerateTaskRun creates a TaskRun with all build phases as sequential steps.
func GenerateTaskRun(
	cfg *config.Config,
	build *buildapi.Build,
	buildRun *buildapi.BuildRun,
	serviceAccountName string,
	strategy buildapi.BuilderStrategy,
) (*pipelineapi.TaskRun, error) {
	generator := NewTaskRunGenerator(cfg, build, buildRun, serviceAccountName, strategy)

	executor, err := GenerateBuildRunExecutor(build, buildRun, strategy, generator)
	if err != nil {
		return nil, err
	}

	taskRun, ok := executor.(*pipelineapi.TaskRun)
	if !ok {
		return nil, fmt.Errorf("expected TaskRun but got %T", executor)
	}

	return taskRun, nil
}

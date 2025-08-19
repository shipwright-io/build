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

// GeneratePipelineRun creates a Tekton PipelineRun object from a Build and BuildRun.
// It generates a TaskRun, and then embeds the TaskSpec into a PipelineRun.
func GeneratePipelineRun(cfg *config.Config, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun, serviceAccountName string, strategy buildv1beta1.BuilderStrategy) (*pipelineapi.PipelineRun, error) {
	// Generate a TaskRun object using the existing logic
	taskRun, err := GenerateTaskRun(cfg, build, buildRun, serviceAccountName, strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to generate TaskRun: %w", err)
	}

	// Extract workspace bindings from the TaskSpec workspaces
	var workspaceBindings []pipelineapi.WorkspacePipelineTaskBinding
	for _, workspace := range taskRun.Spec.TaskSpec.Workspaces {
		workspaceBindings = append(workspaceBindings, pipelineapi.WorkspacePipelineTaskBinding{
			Name:      workspace.Name,
			Workspace: workspace.Name,
		})
	}

	// Create the PipelineRun and embed the TaskSpec from the generated TaskRun
	pipelineRun := &pipelineapi.PipelineRun{
		ObjectMeta: taskRun.ObjectMeta,
		Spec: pipelineapi.PipelineRunSpec{
			PipelineSpec: &pipelineapi.PipelineSpec{
				Params: taskRun.Spec.TaskSpec.Params,
				Tasks: []pipelineapi.PipelineTask{
					{
						Name: "build", // required field for the embedded task
						TaskSpec: &pipelineapi.EmbeddedTask{
							TaskSpec: *taskRun.Spec.TaskSpec,
						},
						Params:     taskRun.Spec.Params,
						Workspaces: workspaceBindings,
					},
				},
			},
			TaskRunTemplate: pipelineapi.PipelineTaskRunTemplate{
				ServiceAccountName: taskRun.Spec.ServiceAccountName,
				PodTemplate:        taskRun.Spec.PodTemplate,
			},
			Workspaces: taskRun.Spec.Workspaces,
			Params:     taskRun.Spec.Params,
			Timeouts: &pipelineapi.TimeoutFields{
				Pipeline: taskRun.Spec.Timeout,
			},
		},
	}

	return pipelineRun, nil
}

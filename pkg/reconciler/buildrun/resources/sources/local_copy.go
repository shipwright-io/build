// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package sources

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/shipwright-io/build/pkg/config"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

// WaiterContainerName name given to the container watier container.
const WaiterContainerName = "source-local"

// AppendLocalCopyStep defines and append a new task based on the waiter container template, passed
// by the configuration instance.
func AppendLocalCopyStep(cfg *config.Config, taskSpec *pipelineapi.TaskSpec, timeout *metav1.Duration) {
	step := pipelineapi.Step{
		// the data upload mechanism targets a specific POD, and in this POD it aims for a specific
		// container name, and having a static name, makes this process straight forward.
		Name:             WaiterContainerName,
		Image:            cfg.WaiterContainerTemplate.Image,
		ImagePullPolicy:  cfg.WaiterContainerTemplate.ImagePullPolicy,
		Command:          cfg.WaiterContainerTemplate.Command,
		Args:             cfg.WaiterContainerTemplate.Args,
		Env:              cfg.WaiterContainerTemplate.Env,
		ComputeResources: cfg.WaiterContainerTemplate.Resources,
		SecurityContext:  cfg.WaiterContainerTemplate.SecurityContext,
		WorkingDir:       cfg.WaiterContainerTemplate.WorkingDir,
	}

	if timeout != nil {
		step.Args = append(step.Args, fmt.Sprintf("--timeout=%s", timeout.Duration.String()))
	}
	taskSpec.Steps = append(taskSpec.Steps, step)
}

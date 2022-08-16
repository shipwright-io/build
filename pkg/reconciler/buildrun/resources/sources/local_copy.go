// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package sources

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/shipwright-io/build/pkg/config"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// WaiterContainerName name given to the container watier container.
const WaiterContainerName = "source-local"

// AppendLocalCopyStep defines and append a new task based on the waiter container template, passed
// by the configuration instance.
func AppendLocalCopyStep(cfg *config.Config, taskSpec *tektonv1beta1.TaskSpec, timeout *metav1.Duration) {
	step := *cfg.WaiterContainerTemplate.DeepCopy()
	// the data upload mechanism targets a specific POD, and in this POD it aims for a specific
	// container name, and having a static name, makes this process straight forward.
	step.Name = WaiterContainerName

	if timeout != nil {
		step.Args = append(step.Args, fmt.Sprintf("--timeout=%s", timeout.Duration.String()))
	}
	taskSpec.Steps = append(taskSpec.Steps, step)
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"fmt"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// RemoteArtifactsContainerName name for the container dealing with remote artifacts download.
const RemoteArtifactsContainerName = "sources-http"

// AppendHTTPStep appends the step for a HTTP source to the TaskSpec
func AppendHTTPStep(
	cfg *config.Config,
	taskSpec *tektonv1beta1.TaskSpec,
	source buildv1alpha1.BuildSource,
) {
	// HTTP is done currently all in a single step, see if there is already one
	httpStep := findExistingHTTPSourcesStep(taskSpec)
	if httpStep != nil {
		httpStep.Args[3] = fmt.Sprintf("%s ; wget %q", httpStep.Args[3], source.URL)
	} else {
		httpStep := tektonv1beta1.Step{
			Name:       RemoteArtifactsContainerName,
			Image:      cfg.RemoteArtifactsContainerImage,
			WorkingDir: fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramSourceRoot),
			Command: []string{
				"/bin/sh",
			},
			Args: []string{
				"-e",
				"-x",
				"-c",
				fmt.Sprintf("wget %q", source.URL),
			},
		}

		// append the git step
		taskSpec.Steps = append(taskSpec.Steps, httpStep)
	}
}

func findExistingHTTPSourcesStep(taskSpec *tektonv1beta1.TaskSpec) *tektonv1beta1.Step {
	for _, candidateStep := range taskSpec.Steps {
		if candidateStep.Name == RemoteArtifactsContainerName {
			return &candidateStep
		}
	}
	return nil
}

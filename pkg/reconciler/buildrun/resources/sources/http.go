// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"fmt"
	"path/filepath"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// AppendHTTPStep appends the step for a HTTP source to the TaskSpec
func AppendHTTPStep(
	cfg *config.Config,
	taskSpec *tektonv1beta1.TaskSpec,
	source buildv1alpha1.BuildSource,
) error {
	if source.HTTP == nil {
		return fmt.Errorf("http information for source %s is not specified", source.Name)
	}

	httpStep := tektonv1beta1.Step{
		Container: corev1.Container{
			Name:  fmt.Sprintf("source-%s", source.Name),
			Image: cfg.RemoteArtifactsContainerImage,
			WorkingDir: filepath.Join(
				fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramSourceRoot),
				source.Destination),
			Command: []string{
				"/bin/sh",
			},
			Args: []string{
				"-e",
				"-x",
				"-c",
				fmt.Sprintf("wget \"%s\"", source.HTTP.URL),
			},
		},
	}

	// append the git step
	taskSpec.Steps = append(taskSpec.Steps, httpStep)
	return nil
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"fmt"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

func AppendGitStep(
	cfg *config.Config,
	taskSpec *tektonv1beta1.TaskSpec,
	source buildv1alpha1.Source,
	name string,
) {
	// append the result
	taskSpec.Results = append(taskSpec.Results, tektonv1beta1.TaskResult{
		Name:        fmt.Sprintf("%ssource-%s-commit-sha", prefixParamsResultsVolumes, name),
		Description: "The commit SHA of the cloned source.",
	})

	// initialize the step
	gitStep := tektonv1beta1.Step{
		Container: corev1.Container{
			Name:  fmt.Sprintf("source-%s", name),
			Image: cfg.GitContainerImage,
			Command: []string{
				"/ko-app/git",
			},
			Args: []string{
				"--url",
				source.URL,
				"--target",
				fmt.Sprintf("$(params.%s%s)", prefixParamsResultsVolumes, paramSourceRoot),
				"--result-file-commit-sha",
				fmt.Sprintf("$(results.%ssource-%s-commit-sha.path)", prefixParamsResultsVolumes, name),
			},
			SecurityContext: &corev1.SecurityContext{
				RunAsUser:  nonRoot,
				RunAsGroup: nonRoot,
			},

			// TODO Resources
		},
	}

	// Check if a revision is defined
	if source.Revision != nil {
		// append the argument
		gitStep.Container.Args = append(
			gitStep.Container.Args,
			"--revision",
			*source.Revision,
		)
	}

	if source.Credentials != nil {
		// ensure the value is there
		AppendSecretVolume(taskSpec, source.Credentials.Name)

		// define the volume mount on the container
		gitStep.VolumeMounts = append(gitStep.VolumeMounts, corev1.VolumeMount{
			Name:      prefixParamsResultsVolumes + source.Credentials.Name,
			MountPath: "/workspace/source-secret",
			ReadOnly:  true,
		})

		// append the argument
		gitStep.Container.Args = append(
			gitStep.Container.Args,
			"--secret-path",
			"/workspace/source-secret",
		)
	}

	// append the git step
	taskSpec.Steps = append(taskSpec.Steps, gitStep)
}

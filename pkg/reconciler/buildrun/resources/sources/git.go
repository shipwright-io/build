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

// AppendGitSourceStep appends the Git step and results and volume if needed to the TaskSpec
func AppendGitSourceStep(
	cfg *config.Config,
	taskSpec *tektonv1beta1.TaskSpec,
	source buildv1alpha1.Source,
	name string,
) {
	revision := ""
	if source.Revision != nil {
		revision = *source.Revision
	}
	internalAppendGitStep(cfg, taskSpec, name, source.URL, revision, "", source.Credentials)
}

func AppendGitStep(cfg *config.Config, taskSpec *tektonv1beta1.TaskSpec, source buildv1alpha1.BuildSource) error {
	if source.Git == nil {
		return fmt.Errorf("git source did not have any source information specified")
	}
	internalAppendGitStep(cfg, taskSpec, source.Name, source.Git.URL, source.Git.Revision, source.Destination, source.Git.Credentials)
	return nil
}

func internalAppendGitStep(cfg *config.Config, taskSpec *tektonv1beta1.TaskSpec, name, url, revision, destination string, credentials *corev1.LocalObjectReference) {
	// append the result
	taskSpec.Results = append(taskSpec.Results, tektonv1beta1.TaskResult{
		Name:        fmt.Sprintf("%s-source-%s-commit-sha", prefixParamsResultsVolumes, name),
		Description: "The commit SHA of the cloned source.",
	})

	// initialize the step from the template
	gitStep := tektonv1beta1.Step{
		Container: *cfg.GitContainerTemplate.DeepCopy(),
	}

	// add the build-specific details
	gitStep.Container.Name = fmt.Sprintf("source-%s", name)
	gitStep.Container.Args = []string{
		"--url",
		url,
		"--target",
		filepath.Join(fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramSourceRoot),
			destination),
		"--result-file-commit-sha",
		fmt.Sprintf("$(results.%s-source-%s-commit-sha.path)", prefixParamsResultsVolumes, name),
	}

	// Check if a revision is defined
	if len(revision) > 0 {
		// append the argument
		gitStep.Container.Args = append(
			gitStep.Container.Args,
			"--revision",
			revision,
		)
	}

	if credentials != nil {
		// ensure the value is there
		AppendSecretVolume(taskSpec, credentials.Name)

		secretMountPath := fmt.Sprintf("/workspace/%s-source-secret", prefixParamsResultsVolumes)

		// define the volume mount on the container
		gitStep.VolumeMounts = append(gitStep.VolumeMounts, corev1.VolumeMount{
			Name:      SanitizeVolumeNameForSecretName(credentials.Name),
			MountPath: secretMountPath,
			ReadOnly:  true,
		})

		// append the argument
		gitStep.Container.Args = append(
			gitStep.Container.Args,
			"--secret-path",
			secretMountPath,
		)
	}

	// append the git step
	taskSpec.Steps = append(taskSpec.Steps, gitStep)
}

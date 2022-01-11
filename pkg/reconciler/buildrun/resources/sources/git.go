// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"fmt"
	"strings"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

const (
	commitSHAResult    = "commit-sha"
	commitAuthorResult = "commit-author"
	branchName         = "branch-name"
)

// AppendGitStep appends the Git step and results and volume if needed to the TaskSpec
func AppendGitStep(
	cfg *config.Config,
	taskSpec *tektonv1beta1.TaskSpec,
	source buildv1alpha1.Source,
	name string,
) {
	// append the result
	taskSpec.Results = append(taskSpec.Results, tektonv1beta1.TaskResult{
		Name:        fmt.Sprintf("%s-source-%s-%s", prefixParamsResultsVolumes, name, commitSHAResult),
		Description: "The commit SHA of the cloned source.",
	}, tektonv1beta1.TaskResult{
		Name:        fmt.Sprintf("%s-source-%s-%s", prefixParamsResultsVolumes, name, commitAuthorResult),
		Description: "The author of the last commit of the cloned source.",
	}, tektonv1beta1.TaskResult{
		Name:        fmt.Sprintf("%s-source-%s-%s", prefixParamsResultsVolumes, name, branchName),
		Description: "The name of the branch used of the cloned source.",
	})

	// initialize the step from the template
	gitStep := tektonv1beta1.Step{
		Container: *cfg.GitContainerTemplate.DeepCopy(),
	}

	// add the build-specific details
	gitStep.Container.Name = fmt.Sprintf("source-%s", name)
	gitStep.Container.Args = []string{
		"--url",
		source.URL,
		"--target",
		fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramSourceRoot),
		"--result-file-commit-sha",
		fmt.Sprintf("$(results.%s-source-%s-%s.path)", prefixParamsResultsVolumes, name, commitSHAResult),
		"--result-file-commit-author",
		fmt.Sprintf("$(results.%s-source-%s-%s.path)", prefixParamsResultsVolumes, name, commitAuthorResult),
		"--result-file-branch-name",
		fmt.Sprintf("$(results.%s-source-%s-%s.path)", prefixParamsResultsVolumes, name, branchName),
		"--result-file-error-message",
		fmt.Sprintf("$(results.%s-error-message.path)", prefixParamsResultsVolumes),
		"--result-file-error-reason",
		fmt.Sprintf("$(results.%s-error-reason.path)", prefixParamsResultsVolumes),
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

	// If configure, use Git URL rewrite flag
	if cfg.GitRewriteRule {
		gitStep.Container.Args = append(gitStep.Container.Args, "--git-url-rewrite")
	}

	if source.Credentials != nil {
		// ensure the value is there
		AppendSecretVolume(taskSpec, source.Credentials.Name)

		secretMountPath := fmt.Sprintf("/workspace/%s-source-secret", prefixParamsResultsVolumes)

		// define the volume mount on the container
		gitStep.VolumeMounts = append(gitStep.VolumeMounts, corev1.VolumeMount{
			Name:      SanitizeVolumeNameForSecretName(source.Credentials.Name),
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

// AppendGitResult append git source result to build run
func AppendGitResult(buildRun *buildv1alpha1.BuildRun, name string, results []tektonv1beta1.TaskRunResult) {
	commitAuthor := findResultValue(results, fmt.Sprintf("%s-source-%s-%s", prefixParamsResultsVolumes, name, commitAuthorResult))
	commitSha := findResultValue(results, fmt.Sprintf("%s-source-%s-%s", prefixParamsResultsVolumes, name, commitSHAResult))
	branchName := findResultValue(results, fmt.Sprintf("%s-source-%s-%s", prefixParamsResultsVolumes, name, branchName))

	if strings.TrimSpace(commitAuthor) != "" || strings.TrimSpace(commitSha) != "" || strings.TrimSpace(branchName) != "" {
		buildRun.Status.Sources = append(buildRun.Status.Sources, buildv1alpha1.SourceResult{
			Name: name,
			Git: &buildv1alpha1.GitSourceResult{
				CommitAuthor: commitAuthor,
				CommitSha:    commitSha,
				BranchName:   branchName,
			},
		})
	}
}

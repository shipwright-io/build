// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"fmt"
	"strings"

	"github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
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
	taskSpec *pipelineapi.TaskSpec,
	git buildv1beta1.Git,
	name string,
) {
	// append the result
	taskSpec.Results = append(taskSpec.Results, pipelineapi.TaskResult{
		Name:        fmt.Sprintf("%s-source-%s-%s", prefixParamsResultsVolumes, name, commitSHAResult),
		Description: "The commit SHA of the cloned source.",
	}, pipelineapi.TaskResult{
		Name:        fmt.Sprintf("%s-source-%s-%s", prefixParamsResultsVolumes, name, commitAuthorResult),
		Description: "The author of the last commit of the cloned source.",
	}, pipelineapi.TaskResult{
		Name:        fmt.Sprintf("%s-source-%s-%s", prefixParamsResultsVolumes, name, branchName),
		Description: "The name of the branch used of the cloned source.",
	})

	// initialize the step from the template and the build-specific arguments
	gitStep := pipelineapi.Step{
		Name:            fmt.Sprintf("source-%s", name),
		Image:           cfg.GitContainerTemplate.Image,
		ImagePullPolicy: cfg.GitContainerTemplate.ImagePullPolicy,
		Command:         cfg.GitContainerTemplate.Command,
		Args: []string{
			"--url",
			git.URL,
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
		},
		Env:              cfg.GitContainerTemplate.Env,
		ComputeResources: cfg.GitContainerTemplate.Resources,
		SecurityContext:  cfg.GitContainerTemplate.SecurityContext,
		WorkingDir:       cfg.GitContainerTemplate.WorkingDir,
	}

	// Check if a revision is defined
	if git.Revision != nil {
		// append the argument
		gitStep.Args = append(
			gitStep.Args,
			"--revision",
			*git.Revision,
		)
	}

	// If configure, use Git URL rewrite flag
	if cfg.GitRewriteRule {
		gitStep.Args = append(gitStep.Args, "--git-url-rewrite")
	}

	if git.CloneSecret != nil {
		// ensure the value is there
		AppendSecretVolume(taskSpec, *git.CloneSecret)

		secretMountPath := fmt.Sprintf("/workspace/%s-source-secret", prefixParamsResultsVolumes)

		// define the volume mount on the container
		gitStep.VolumeMounts = append(gitStep.VolumeMounts, corev1.VolumeMount{
			Name:      SanitizeVolumeNameForSecretName(*git.CloneSecret),
			MountPath: secretMountPath,
			ReadOnly:  true,
		})

		// append the argument
		gitStep.Args = append(
			gitStep.Args,
			"--secret-path",
			secretMountPath,
		)
	}

	// append the git step
	taskSpec.Steps = append(taskSpec.Steps, gitStep)
}

// AppendGitResult append git source result to build run
func AppendGitResult(buildRun *buildv1beta1.BuildRun, name string, results []pipelineapi.TaskRunResult) {
	commitAuthor := findResultValue(results, fmt.Sprintf("%s-source-%s-%s", prefixParamsResultsVolumes, name, commitAuthorResult))
	commitSha := findResultValue(results, fmt.Sprintf("%s-source-%s-%s", prefixParamsResultsVolumes, name, commitSHAResult))
	branchName := findResultValue(results, fmt.Sprintf("%s-source-%s-%s", prefixParamsResultsVolumes, name, branchName))

	if strings.TrimSpace(commitAuthor) != "" || strings.TrimSpace(commitSha) != "" || strings.TrimSpace(branchName) != "" {
		buildRun.Status.Source.Git = &v1beta1.GitSourceResult{
			CommitAuthor: commitAuthor,
			CommitSha:    commitSha,
			BranchName:   branchName,
		}
	}
}

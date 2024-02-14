// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
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
	source buildv1beta1.Git,
	name string,
) {
	// append the result
	taskSpec.Results = append(taskSpec.Results,
		pipelineapi.TaskResult{
			Name:        fmt.Sprintf("%s-source-%s-%s", PrefixParamsResultsVolumes, name, commitSHAResult),
			Description: "The commit SHA of the cloned source.",
		},
		pipelineapi.TaskResult{
			Name:        fmt.Sprintf("%s-source-%s-%s", PrefixParamsResultsVolumes, name, commitAuthorResult),
			Description: "The author of the last commit of the cloned source.",
		},
		pipelineapi.TaskResult{
			Name:        fmt.Sprintf("%s-source-%s-%s", PrefixParamsResultsVolumes, name, branchName),
			Description: "The name of the branch used of the cloned source.",
		},
	)

	// initialize the step from the template and the build-specific arguments
	gitStep := pipelineapi.Step{
		Name:            fmt.Sprintf("source-%s", name),
		Image:           cfg.GitContainerTemplate.Image,
		ImagePullPolicy: cfg.GitContainerTemplate.ImagePullPolicy,
		Command:         cfg.GitContainerTemplate.Command,
		Args: []string{
			"--url", source.URL,
			"--target", fmt.Sprintf("$(params.%s-%s)", PrefixParamsResultsVolumes, paramSourceRoot),
			"--result-file-commit-sha", fmt.Sprintf("$(results.%s-source-%s-%s.path)", PrefixParamsResultsVolumes, name, commitSHAResult),
			"--result-file-commit-author", fmt.Sprintf("$(results.%s-source-%s-%s.path)", PrefixParamsResultsVolumes, name, commitAuthorResult),
			"--result-file-branch-name", fmt.Sprintf("$(results.%s-source-%s-%s.path)", PrefixParamsResultsVolumes, name, branchName),
			"--result-file-error-message", fmt.Sprintf("$(results.%s-error-message.path)", PrefixParamsResultsVolumes),
			"--result-file-error-reason", fmt.Sprintf("$(results.%s-error-reason.path)", PrefixParamsResultsVolumes),
			"--result-file-source-timestamp", fmt.Sprintf("$(results.%s-source-%s-source-timestamp.path)", PrefixParamsResultsVolumes, name),
		},
		Env:              cfg.GitContainerTemplate.Env,
		ComputeResources: cfg.GitContainerTemplate.Resources,
		SecurityContext:  cfg.GitContainerTemplate.SecurityContext,
		WorkingDir:       cfg.GitContainerTemplate.WorkingDir,
	}

	// Check if a revision is defined
	if source.Revision != nil {
		// append the argument
		gitStep.Args = append(
			gitStep.Args,
			"--revision",
			*source.Revision,
		)
	}

	// If configure, use Git URL rewrite flag
	if cfg.GitRewriteRule {
		gitStep.Args = append(gitStep.Args, "--git-url-rewrite")
	}

	if source.CloneSecret != nil {
		// ensure the value is there
		AppendSecretVolume(taskSpec, *source.CloneSecret)

		secretMountPath := fmt.Sprintf("/workspace/%s-source-secret", PrefixParamsResultsVolumes)

		// define the volume mount on the container
		gitStep.VolumeMounts = append(gitStep.VolumeMounts, corev1.VolumeMount{
			Name:      SanitizeVolumeNameForSecretName(*source.CloneSecret),
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
	commitAuthor := FindResultValue(results, name, commitAuthorResult)
	commitSha := FindResultValue(results, name, commitSHAResult)
	branchName := FindResultValue(results, name, branchName)

	if strings.TrimSpace(commitAuthor) != "" || strings.TrimSpace(commitSha) != "" || strings.TrimSpace(branchName) != "" {
		if buildRun.Status.Source == nil {
			buildRun.Status.Source = &build.SourceResult{}
		}
		buildRun.Status.Source.Git = &v1beta1.GitSourceResult{
			CommitAuthor: commitAuthor,
			CommitSha:    commitSha,
			BranchName:   branchName,
		}
	}
}

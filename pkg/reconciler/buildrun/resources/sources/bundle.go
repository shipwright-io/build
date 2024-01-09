// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"fmt"
	"strings"

	core "k8s.io/api/core/v1"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

// AppendBundleStep appends the bundle step to the TaskSpec
func AppendBundleStep(cfg *config.Config, taskSpec *pipelineapi.TaskSpec, oci *build.OCIArtifact, name string) {
	// append the result
	taskSpec.Results = append(taskSpec.Results, pipelineapi.TaskResult{
		Name:        fmt.Sprintf("%s-source-%s-image-digest", prefixParamsResultsVolumes, name),
		Description: "The digest of the bundle image.",
	})

	// initialize the step from the template and the build-specific arguments
	bundleStep := pipelineapi.Step{
		Name:            fmt.Sprintf("source-%s", name),
		Image:           cfg.BundleContainerTemplate.Image,
		ImagePullPolicy: cfg.BundleContainerTemplate.ImagePullPolicy,
		Command:         cfg.BundleContainerTemplate.Command,
		Args: []string{
			"--image", oci.Image,
			"--target", fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramSourceRoot),
			"--result-file-image-digest", fmt.Sprintf("$(results.%s-source-%s-image-digest.path)", prefixParamsResultsVolumes, name),
		},
		Env:              cfg.BundleContainerTemplate.Env,
		ComputeResources: cfg.BundleContainerTemplate.Resources,
		SecurityContext:  cfg.BundleContainerTemplate.SecurityContext,
		WorkingDir:       cfg.BundleContainerTemplate.WorkingDir,
	}

	// add credentials mount, if provided
	if oci.PullSecret != nil {
		AppendSecretVolume(taskSpec, *oci.PullSecret)

		secretMountPath := fmt.Sprintf("/workspace/%s-pull-secret", prefixParamsResultsVolumes)

		// define the volume mount on the container
		bundleStep.VolumeMounts = append(bundleStep.VolumeMounts, core.VolumeMount{
			Name:      SanitizeVolumeNameForSecretName(*oci.PullSecret),
			MountPath: secretMountPath,
			ReadOnly:  true,
		})

		// append the argument
		bundleStep.Args = append(bundleStep.Args,
			"--secret-path", secretMountPath,
		)
	}

	// add prune flag in when prune after pull is configured
	if oci.Prune != nil && *oci.Prune == build.PruneAfterPull {
		bundleStep.Args = append(bundleStep.Args, "--prune")
	}

	taskSpec.Steps = append(taskSpec.Steps, bundleStep)
}

// AppendBundleResult append bundle source result to build run
func AppendBundleResult(buildRun *build.BuildRun, name string, results []pipelineapi.TaskRunResult) {
	imageDigest := findResultValue(results, fmt.Sprintf("%s-source-%s-image-digest", prefixParamsResultsVolumes, name))
	if strings.TrimSpace(imageDigest) != "" {
		buildRun.Status.Source.OciArtifact = &build.OciArtifactSourceResult{
			Digest: imageDigest,
		}
	}
}

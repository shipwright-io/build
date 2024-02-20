// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"fmt"
	"strings"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	corev1 "k8s.io/api/core/v1"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

// AppendBundleStep appends the bundle step to the TaskSpec
func AppendBundleStep(cfg *config.Config, taskSpec *pipelineapi.TaskSpec, oci *build.OCIArtifact, name string) {
	// append the result
	taskSpec.Results = append(taskSpec.Results,
		pipelineapi.TaskResult{
			Name:        fmt.Sprintf("%s-source-%s-image-digest", PrefixParamsResultsVolumes, name),
			Description: "The digest of the bundle image.",
		},
	)

	// initialize the step from the template and the build-specific arguments
	bundleStep := pipelineapi.Step{
		Name:            fmt.Sprintf("source-%s", name),
		Image:           cfg.BundleContainerTemplate.Image,
		ImagePullPolicy: cfg.BundleContainerTemplate.ImagePullPolicy,
		Command:         cfg.BundleContainerTemplate.Command,
		Args: []string{
			"--image", oci.Image,
			"--target", fmt.Sprintf("$(params.%s-%s)", PrefixParamsResultsVolumes, paramSourceRoot),
			"--result-file-image-digest", fmt.Sprintf("$(results.%s-source-%s-image-digest.path)", PrefixParamsResultsVolumes, name),
			"--result-file-source-timestamp", fmt.Sprintf("$(results.%s-source-%s-source-timestamp.path)", PrefixParamsResultsVolumes, name),
		},
		Env:              cfg.BundleContainerTemplate.Env,
		ComputeResources: cfg.BundleContainerTemplate.Resources,
		SecurityContext:  cfg.BundleContainerTemplate.SecurityContext,
		WorkingDir:       cfg.BundleContainerTemplate.WorkingDir,
	}

	// add credentials mount, if provided
	if oci.PullSecret != nil {
		AppendSecretVolume(taskSpec, *oci.PullSecret)

		secretMountPath := fmt.Sprintf("/workspace/%s-pull-secret", PrefixParamsResultsVolumes)

		// define the volume mount on the container
		bundleStep.VolumeMounts = append(bundleStep.VolumeMounts, corev1.VolumeMount{
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
	imageDigest := FindResultValue(results, name, "image-digest")

	if strings.TrimSpace(imageDigest) != "" {
		if buildRun.Status.Source == nil {
			buildRun.Status.Source = &build.SourceResult{}
		}
		buildRun.Status.Source.OciArtifact = &build.OciArtifactSourceResult{
			Digest: imageDigest,
		}
	}
}

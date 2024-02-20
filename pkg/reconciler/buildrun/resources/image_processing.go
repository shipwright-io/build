// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/sources"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	core "k8s.io/api/core/v1"
)

const (
	containerNameImageProcessing = "image-processing"
	outputDirectoryMountPath     = "/workspace/output-image"
	paramOutputDirectory         = "output-directory"
)

// SetupImageProcessing appends the image-processing step to a TaskRun if desired
func SetupImageProcessing(taskRun *pipelineapi.TaskRun, cfg *config.Config, buildOutput, buildRunOutput build.Image) {
	stepArgs := []string{}

	// Check if any build step references the output-directory system parameter. If that is the case,
	// then we assume that Shipwright performs the image push operation.
	volumeAdded := false
	prefixedOutputDirectory := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputDirectory)
	for i := range taskRun.Spec.TaskSpec.Steps {
		step := taskRun.Spec.TaskSpec.Steps[i]

		if isStepReferencingParameter(&step, prefixedOutputDirectory) {
			if !volumeAdded {
				volumeAdded = true

				// add an emptyDir volume for the output directory
				taskRun.Spec.TaskSpec.Volumes = append(taskRun.Spec.TaskSpec.Volumes, core.Volume{
					Name: prefixedOutputDirectory,
					VolumeSource: core.VolumeSource{
						EmptyDir: &core.EmptyDirVolumeSource{},
					},
				})

				// add the parameter definition
				taskRun.Spec.TaskSpec.Params = append(taskRun.Spec.TaskSpec.Params, pipelineapi.ParamSpec{
					Name: prefixedOutputDirectory,
					Type: pipelineapi.ParamTypeString,
				})

				// add the parameter value
				taskRun.Spec.Params = append(taskRun.Spec.Params, pipelineapi.Param{
					Name: prefixedOutputDirectory,
					Value: pipelineapi.ParamValue{
						StringVal: outputDirectoryMountPath,
						Type:      pipelineapi.ParamTypeString,
					},
				})

				// add the push argument to the command
				stepArgs = append(stepArgs, "--push", fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramOutputDirectory))
			}

			// add a volumeMount to the step
			taskRun.Spec.TaskSpec.Steps[i].VolumeMounts = append(taskRun.Spec.TaskSpec.Steps[i].VolumeMounts, core.VolumeMount{
				Name:      prefixedOutputDirectory,
				MountPath: outputDirectoryMountPath,
			})
		}
	}

	// check if we need to set image labels
	annotations := mergeMaps(buildOutput.Annotations, buildRunOutput.Annotations)
	if len(annotations) > 0 {
		stepArgs = append(stepArgs, convertMutateArgs("--annotation", annotations)...)
	}

	// check if we need to set image labels
	labels := mergeMaps(buildOutput.Labels, buildRunOutput.Labels)
	if len(labels) > 0 {
		stepArgs = append(stepArgs, convertMutateArgs("--label", labels)...)
	}

	// check if there is anything to do
	if len(stepArgs) > 0 {
		// add the image argument
		stepArgs = append(stepArgs, "--image", fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramOutputImage))

		// add the insecure flag
		stepArgs = append(stepArgs, fmt.Sprintf("--insecure=$(params.%s-%s)", prefixParamsResultsVolumes, paramOutputInsecure))

		// add the result arguments
		stepArgs = append(stepArgs, "--result-file-image-digest", fmt.Sprintf("$(results.%s-%s.path)", prefixParamsResultsVolumes, imageDigestResult))
		stepArgs = append(stepArgs, "--result-file-image-size", fmt.Sprintf("$(results.%s-%s.path)", prefixParamsResultsVolumes, imageSizeResult))

		// add the push step

		// initialize the step from the template and the build-specific arguments
		imageProcessingStep := pipelineapi.Step{
			Name:             containerNameImageProcessing,
			Image:            cfg.ImageProcessingContainerTemplate.Image,
			ImagePullPolicy:  cfg.ImageProcessingContainerTemplate.ImagePullPolicy,
			Command:          cfg.ImageProcessingContainerTemplate.Command,
			Args:             stepArgs,
			Env:              cfg.ImageProcessingContainerTemplate.Env,
			ComputeResources: cfg.ImageProcessingContainerTemplate.Resources,
			SecurityContext:  cfg.ImageProcessingContainerTemplate.SecurityContext,
			WorkingDir:       cfg.ImageProcessingContainerTemplate.WorkingDir,
		}

		if volumeAdded {
			imageProcessingStep.VolumeMounts = append(imageProcessingStep.VolumeMounts, core.VolumeMount{
				Name:      prefixedOutputDirectory,
				MountPath: outputDirectoryMountPath,
				ReadOnly:  true,
			})
		}

		if buildOutput.PushSecret != nil {
			sources.AppendSecretVolume(taskRun.Spec.TaskSpec, *buildOutput.PushSecret)

			secretMountPath := fmt.Sprintf("/workspace/%s-push-secret", prefixParamsResultsVolumes)

			// define the volume mount on the container
			imageProcessingStep.VolumeMounts = append(imageProcessingStep.VolumeMounts, core.VolumeMount{
				Name:      sources.SanitizeVolumeNameForSecretName(*buildOutput.PushSecret),
				MountPath: secretMountPath,
				ReadOnly:  true,
			})

			// append the argument
			imageProcessingStep.Args = append(imageProcessingStep.Args,
				"--secret-path", secretMountPath,
			)
		}

		// append the mutate step
		taskRun.Spec.TaskSpec.Steps = append(taskRun.Spec.TaskSpec.Steps, imageProcessingStep)
	}
}

// convertMutateArgs to convert the argument map to comma separated values
func convertMutateArgs(flag string, args map[string]string) []string {
	var result []string

	for key, value := range args {
		result = append(result, flag, fmt.Sprintf("%s=%s", key, value))
	}

	return result
}

// mergeMaps takes 2 maps as input and merge the second into the first
// values in second would takes precedence if both maps have same keys
func mergeMaps(first map[string]string, second map[string]string) map[string]string {
	if len(first) == 0 {
		first = map[string]string{}
	}
	for k, v := range second {
		first[k] = v
	}
	return first
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	core "k8s.io/api/core/v1"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/sources"
	"github.com/spf13/pflag"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const (
	containerNameImageProcessing = "image-processing"
	outputDirectoryMountPath     = "/workspace/output-image"
	paramOutputDirectory         = "output-directory"
)

type VulnerablilityScanParams struct {
	build.VulnerabilityScanOptions
}

var _ pflag.Value = &VulnerablilityScanParams{}

func (v *VulnerablilityScanParams) Set(s string) error {
	return json.Unmarshal([]byte(s), v)
}

func (v *VulnerablilityScanParams) String() string {
	data, err := json.Marshal(*v)
	if err != nil {
		panic(err.Error())
	}
	return string(data)
}

func (v *VulnerablilityScanParams) Type() string {
	return "vulnerability-scan-params"
}

// SetupImageProcessing appends the image-processing step to a TaskRun if desired
func SetupImageProcessing(taskRun *pipelineapi.TaskRun, cfg *config.Config, creationTimestamp time.Time, buildOutput, buildRunOutput build.Image) error {
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

	vulnerabilitySettings := GetVulnerabilityScanOptions(buildOutput, buildRunOutput)

	// check if we need to add vulnerability scan arguments
	if vulnerabilitySettings != nil && vulnerabilitySettings.Enabled {
		vulnerablilityScanParams := &VulnerablilityScanParams{*vulnerabilitySettings}
		stepArgs = append(stepArgs, "--vuln-settings", vulnerablilityScanParams.String())

		if cfg.VulnerabilityCountLimit > 0 {
			stepArgs = append(stepArgs, "--vuln-count-limit", strconv.Itoa(cfg.VulnerabilityCountLimit))
		}
	}

	// check if we need to set image timestamp
	if imageTimestamp := getImageTimestamp(buildOutput, buildRunOutput); imageTimestamp != nil {
		switch *imageTimestamp {
		case build.OutputImageZeroTimestamp:
			stepArgs = append(stepArgs, "--image-timestamp", "0")

		case build.OutputImageSourceTimestamp:
			if !hasTaskSpecResult(taskRun, "shp-source-default-source-timestamp") {
				return fmt.Errorf("cannot use SourceTimestamp setting, because there is no source timestamp available for this source")
			}

			stepArgs = append(stepArgs, "--image-timestamp-file", "$(results.shp-source-default-source-timestamp.path)")

		case build.OutputImageBuildTimestamp:
			stepArgs = append(stepArgs, "--image-timestamp", strconv.FormatInt(creationTimestamp.Unix(), 10))

		default:
			if _, err := strconv.ParseInt(*imageTimestamp, 10, 64); err != nil {
				return fmt.Errorf("cannot parse output timestamp %s as a number, must be %s, %s, %s, or a an integer",
					*imageTimestamp, build.OutputImageZeroTimestamp, build.OutputImageSourceTimestamp, build.OutputImageBuildTimestamp)
			}

			stepArgs = append(stepArgs, "--image-timestamp", *imageTimestamp)
		}
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
		stepArgs = append(stepArgs, "--result-file-image-vulnerabilities", fmt.Sprintf("$(results.%s-%s.path)", prefixParamsResultsVolumes, imageVulnerabilities))

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

		sources.AppendWriteableVolumes(taskRun.Spec.TaskSpec, &imageProcessingStep, cfg.ContainersWritableDir.WritableHomeDir)

		taskRun.Spec.TaskSpec.Volumes = append(taskRun.Spec.TaskSpec.Volumes, core.Volume{
			Name: "trivy-cache-data",
			VolumeSource: core.VolumeSource{
				EmptyDir: &core.EmptyDirVolumeSource{},
			},
		})
		imageProcessingStep.VolumeMounts = append(imageProcessingStep.VolumeMounts, core.VolumeMount{
			Name:      "trivy-cache-data",
			MountPath: cfg.ContainersWritableDir.TrivyCacheDir,
		})

		imageProcessingStep.Env = append(imageProcessingStep.Env, core.EnvVar{
			Name:  "TRIVY_CACHE_DIR",
			Value: cfg.ContainersWritableDir.TrivyCacheDir,
		})
		// append the mutate step
		taskRun.Spec.TaskSpec.Steps = append(taskRun.Spec.TaskSpec.Steps, imageProcessingStep)
	}

	return nil
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

func GetVulnerabilityScanOptions(buildOutput, buildRunOutput build.Image) *build.VulnerabilityScanOptions {
	switch {
	case buildRunOutput.VulnerabilityScan != nil:
		return buildRunOutput.VulnerabilityScan
	case buildOutput.VulnerabilityScan != nil:
		return buildOutput.VulnerabilityScan
	default:
		return nil
	}
}

func getImageTimestamp(buildOutput, buildRunOutput build.Image) *string {
	switch {
	case buildRunOutput.Timestamp != nil:
		return buildRunOutput.Timestamp

	case buildOutput.Timestamp != nil:
		return buildOutput.Timestamp

	default:
		return nil
	}
}

func hasTaskSpecResult(taskRun *pipelineapi.TaskRun, name string) bool {
	for _, result := range taskRun.Spec.TaskSpec.Results {
		if result.Name == name {
			return true
		}
	}

	return false
}

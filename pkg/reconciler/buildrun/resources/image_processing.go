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

// SetupOutputDirectory scans existing steps for output-directory parameter references
// and sets up the necessary volume, parameter, and volume mounts.
// Returns true if output directory was added.
func SetupOutputDirectory(taskSpec *pipelineapi.TaskSpec, taskRunParams *[]pipelineapi.Param) bool {
	volumeAdded := false
	prefixedOutputDirectory := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputDirectory)

	for i := range taskSpec.Steps {
		step := taskSpec.Steps[i]

		if isStepReferencingParameter(&step, prefixedOutputDirectory) {
			if !volumeAdded {
				volumeAdded = true

				// Add emptyDir volume
				taskSpec.Volumes = append(taskSpec.Volumes, core.Volume{
					Name: prefixedOutputDirectory,
					VolumeSource: core.VolumeSource{
						EmptyDir: &core.EmptyDirVolumeSource{},
					},
				})

				// Add parameter definition
				taskSpec.Params = append(taskSpec.Params, pipelineapi.ParamSpec{
					Name: prefixedOutputDirectory,
					Type: pipelineapi.ParamTypeString,
				})

				// Add parameter value (only for TaskRun)
				if taskRunParams != nil {
					*taskRunParams = append(*taskRunParams, pipelineapi.Param{
						Name: prefixedOutputDirectory,
						Value: pipelineapi.ParamValue{
							StringVal: outputDirectoryMountPath,
							Type:      pipelineapi.ParamTypeString,
						},
					})
				}
			}

			// Add volumeMount to the step
			taskSpec.Steps[i].VolumeMounts = append(taskSpec.Steps[i].VolumeMounts, core.VolumeMount{
				Name:      prefixedOutputDirectory,
				MountPath: outputDirectoryMountPath,
			})
		}
	}

	return volumeAdded
}

// BuildImageProcessingArgs builds the argument list for the image-processing step
// based on output configuration (labels, annotations, vulnerability scanning, timestamps).
func BuildImageProcessingArgs(
	cfg *config.Config,
	creationTimestamp time.Time,
	buildOutput, buildRunOutput build.Image,
	hasOutputDirectory bool,
	hasSourceTimestamp bool,
) ([]string, error) {
	stepArgs := []string{}

	// Add push arg if output directory is used
	if hasOutputDirectory {
		stepArgs = append(stepArgs, "--push", fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramOutputDirectory))
	}

	annotations := mergeMaps(buildOutput.Annotations, buildRunOutput.Annotations)
	if len(annotations) > 0 {
		stepArgs = append(stepArgs, convertMutateArgs("--annotation", annotations)...)
	}

	labels := mergeMaps(buildOutput.Labels, buildRunOutput.Labels)
	if len(labels) > 0 {
		stepArgs = append(stepArgs, convertMutateArgs("--label", labels)...)
	}

	vulnerabilitySettings := GetVulnerabilityScanOptions(buildOutput, buildRunOutput)
	if vulnerabilitySettings != nil && vulnerabilitySettings.Enabled {
		vulnerablilityScanParams := &VulnerablilityScanParams{*vulnerabilitySettings}
		stepArgs = append(stepArgs, "--vuln-settings", vulnerablilityScanParams.String())

		if cfg.VulnerabilityCountLimit > 0 {
			stepArgs = append(stepArgs, "--vuln-count-limit", strconv.Itoa(cfg.VulnerabilityCountLimit))
		}
	}

	if imageTimestamp := getImageTimestamp(buildOutput, buildRunOutput); imageTimestamp != nil {
		switch *imageTimestamp {
		case build.OutputImageZeroTimestamp:
			stepArgs = append(stepArgs, "--image-timestamp", "0")

		case build.OutputImageSourceTimestamp:
			if !hasSourceTimestamp {
				return nil, fmt.Errorf("cannot use SourceTimestamp setting, because there is no source timestamp available for this source")
			}
			stepArgs = append(stepArgs, "--image-timestamp-file", "$(results.shp-source-default-source-timestamp.path)")

		case build.OutputImageBuildTimestamp:
			stepArgs = append(stepArgs, "--image-timestamp", strconv.FormatInt(creationTimestamp.Unix(), 10))

		default:
			if _, err := strconv.ParseInt(*imageTimestamp, 10, 64); err != nil {
				return nil, fmt.Errorf("cannot parse output timestamp %s as a number, must be %s, %s, %s, or a an integer",
					*imageTimestamp, build.OutputImageZeroTimestamp, build.OutputImageSourceTimestamp, build.OutputImageBuildTimestamp)
			}

			stepArgs = append(stepArgs, "--image-timestamp", *imageTimestamp)
		}
	}

	if len(stepArgs) > 0 {
		stepArgs = append(stepArgs, "--image", fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramOutputImage))
		stepArgs = append(stepArgs, fmt.Sprintf("--insecure=$(params.%s-%s)", prefixParamsResultsVolumes, paramOutputInsecure))
		stepArgs = append(stepArgs, "--result-file-image-digest", fmt.Sprintf("$(results.%s-%s.path)", prefixParamsResultsVolumes, imageDigestResult))
		stepArgs = append(stepArgs, "--result-file-image-size", fmt.Sprintf("$(results.%s-%s.path)", prefixParamsResultsVolumes, imageSizeResult))
		stepArgs = append(stepArgs, "--result-file-image-vulnerabilities", fmt.Sprintf("$(results.%s-%s.path)", prefixParamsResultsVolumes, imageVulnerabilities))
	}

	return stepArgs, nil
}

func CreateImageProcessingStep(
	cfg *config.Config,
	taskSpec *pipelineapi.TaskSpec,
	stepArgs []string,
	hasOutputDirectory bool,
	pushSecret *string,
) error {
	if len(stepArgs) == 0 {
		return nil
	}

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

	if hasOutputDirectory {
		prefixedOutputDirectory := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputDirectory)
		imageProcessingStep.VolumeMounts = append(imageProcessingStep.VolumeMounts, core.VolumeMount{
			Name:      prefixedOutputDirectory,
			MountPath: outputDirectoryMountPath,
			ReadOnly:  true,
		})
	}

	if pushSecret != nil {
		sources.AppendSecretVolume(taskSpec, *pushSecret)

		secretMountPath := fmt.Sprintf("/workspace/%s-push-secret", prefixParamsResultsVolumes)

		imageProcessingStep.VolumeMounts = append(imageProcessingStep.VolumeMounts, core.VolumeMount{
			Name:      sources.SanitizeVolumeNameForSecretName(*pushSecret),
			MountPath: secretMountPath,
			ReadOnly:  true,
		})

		imageProcessingStep.Args = append(imageProcessingStep.Args,
			"--secret-path", secretMountPath,
		)
	}

	taskSpec.Volumes = append(taskSpec.Volumes, core.Volume{
		Name: "shp-trivy-cache-data",
		VolumeSource: core.VolumeSource{
			EmptyDir: &core.EmptyDirVolumeSource{},
		},
	})
	imageProcessingStep.VolumeMounts = append(imageProcessingStep.VolumeMounts, core.VolumeMount{
		Name:      "shp-trivy-cache-data",
		MountPath: "/trivy-cache-data",
	})

	imageProcessingStep.Env = append(imageProcessingStep.Env, core.EnvVar{
		Name:  "TRIVY_CACHE_DIR",
		Value: "/trivy-cache-data",
	})

	sources.SetupHomeAndTmpVolumes(taskSpec, &imageProcessingStep)
	taskSpec.Steps = append(taskSpec.Steps, imageProcessingStep)

	return nil
}

// SetupImageProcessing configures image processing for TaskRun execution.
func SetupImageProcessing(taskRun *pipelineapi.TaskRun, cfg *config.Config, creationTimestamp time.Time, buildOutput, buildRunOutput build.Image) error {
	params := []pipelineapi.Param(taskRun.Spec.Params)
	hasOutputDirectory := SetupOutputDirectory(taskRun.Spec.TaskSpec, &params)
	taskRun.Spec.Params = pipelineapi.Params(params)

	hasSourceTimestamp := hasTaskSpecResult(taskRun, "shp-source-default-source-timestamp")
	stepArgs, err := BuildImageProcessingArgs(
		cfg,
		creationTimestamp,
		buildOutput,
		buildRunOutput,
		hasOutputDirectory,
		hasSourceTimestamp,
	)
	if err != nil {
		return err
	}

	return CreateImageProcessingStep(
		cfg,
		taskRun.Spec.TaskSpec,
		stepArgs,
		hasOutputDirectory,
		buildOutput.PushSecret,
	)
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

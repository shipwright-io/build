// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"
	"strconv"
	"strings"

	taskv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
)

const (
	inputSourceResourceName = "source"
	inputGitSourceURL       = "url"
	inputGitSourceRevision  = "revision"
	inputParamBuilderImage  = "BUILDER_IMAGE"
	inputParamDockerfile    = "DOCKERFILE"
	inputParamContextDir    = "CONTEXT_DIR"
	outputImageResourceName = "image"
	outputImageResourceURL  = "url"
)

// getStringTransformations gets us MANDATORY replacements using
// a poor man's templating mechanism - TODO: Use golang templating
func getStringTransformations(fullText string) string {

	stringTransformations := map[string]string{
		"$(build.output.image)":      "$(outputs.resources.image.url)",
		"$(build.builder.image)":     fmt.Sprintf("$(inputs.params.%s)", inputParamBuilderImage),
		"$(build.dockerfile)":        fmt.Sprintf("$(inputs.params.%s)", inputParamDockerfile),
		"$(build.source.contextDir)": fmt.Sprintf("$(inputs.params.%s)", inputParamContextDir),
	}

	// Run the text through all possible replacements
	for k, v := range stringTransformations {
		fullText = strings.ReplaceAll(fullText, k, v)
	}
	return fullText
}

// GenerateTaskSpec creates Tekton TaskRun spec to be used for a build run
func GenerateTaskSpec(
	cfg *config.Config,
	build *buildv1alpha1.Build,
	buildRun *buildv1alpha1.BuildRun,
	buildSteps []buildv1alpha1.BuildStep,
) (*v1beta1.TaskSpec, error) {

	generatedTaskSpec := v1beta1.TaskSpec{
		Resources: &v1beta1.TaskResources{
			Inputs: []v1beta1.TaskResource{
				{
					ResourceDeclaration: taskv1.ResourceDeclaration{
						Name: inputSourceResourceName,
						Type: taskv1.PipelineResourceTypeGit,
					},
				},
			},
			Outputs: []v1beta1.TaskResource{
				{
					ResourceDeclaration: taskv1.ResourceDeclaration{
						Name: outputImageResourceName, // mapped from {{ .Build.OutputImage }}
						Type: taskv1.PipelineResourceTypeImage,
					},
				},
			},
		},
		Params: []v1beta1.ParamSpec{
			{
				Description: "Path to the Dockerfile",
				Name:        inputParamDockerfile,
				Default: &v1beta1.ArrayOrString{
					Type:      v1beta1.ParamTypeString,
					StringVal: "Dockerfile",
				},
			},
			{
				// CONTEXT_DIR comes from the git source specification
				// in the Build object
				Description: "The root of the code",
				Name:        inputParamContextDir,
				Default: &v1beta1.ArrayOrString{
					Type:      v1beta1.ParamTypeString,
					StringVal: ".",
				},
			},
		},
		Steps: []v1beta1.Step{},
	}

	if build.Spec.BuilderImage != nil {
		InputBuilderImage := v1beta1.ParamSpec{
			Description: "Image containing the build tools/logic",
			Name:        inputParamBuilderImage,
			Default: &v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: build.Spec.BuilderImage.ImageURL,
			},
		}
		generatedTaskSpec.Params = append(generatedTaskSpec.Params, InputBuilderImage)
	}

	var vols []corev1.Volume

	for _, containerValue := range buildSteps {

		var taskCommand []string
		for _, buildStrategyCommandPart := range containerValue.Command {
			taskCommand = append(taskCommand, getStringTransformations(buildStrategyCommandPart))
		}

		var taskArgs []string
		for _, buildStrategyArgPart := range containerValue.Args {
			taskArgs = append(taskArgs, getStringTransformations(buildStrategyArgPart))
		}

		taskImage := getStringTransformations(containerValue.Image)

		step := v1beta1.Step{
			Container: corev1.Container{
				Image:           taskImage,
				Name:            containerValue.Name,
				VolumeMounts:    containerValue.VolumeMounts,
				Command:         taskCommand,
				Args:            taskArgs,
				SecurityContext: containerValue.SecurityContext,
				WorkingDir:      containerValue.WorkingDir,
				Resources:       containerValue.Resources,
				Env:             containerValue.Env,
			},
		}

		generatedTaskSpec.Steps = append(generatedTaskSpec.Steps, step)

		// Get volumeMounts added to Task's spec.Volumes
		for _, volumeInBuildStrategy := range containerValue.VolumeMounts {
			newVolume := true
			for _, volumeInTask := range vols {
				if volumeInTask.Name == volumeInBuildStrategy.Name {
					newVolume = false
				}
			}
			if newVolume {
				vols = append(vols, corev1.Volume{
					Name: volumeInBuildStrategy.Name,
				})
			}

		}
	}

	generatedTaskSpec.Volumes = vols

	// checking for runtime-image settings, and appending more steps to the strategy
	if IsRuntimeDefined(build) {
		if err := AmendTaskSpecWithRuntimeImage(cfg, &generatedTaskSpec, build); err != nil {
			return nil, err
		}
	}

	return &generatedTaskSpec, nil
}

// GenerateTaskRun creates a Tekton TaskRun to be used for a build run
func GenerateTaskRun(
	cfg *config.Config,
	build *buildv1alpha1.Build,
	buildRun *buildv1alpha1.BuildRun,
	serviceAccountName string,
	strategy buildv1alpha1.BuilderStrategy,
) (*v1beta1.TaskRun, error) {

	revision := "master"
	if build.Spec.Source.Revision != nil {
		revision = *build.Spec.Source.Revision
	}

	// retrieve expected imageURL form build or buildRun
	var ImageURL string
	if buildRun.Spec.Output != nil {
		ImageURL = buildRun.Spec.Output.ImageURL
	} else {
		ImageURL = build.Spec.Output.ImageURL
	}

	taskSpec, err := GenerateTaskSpec(cfg, build, buildRun, strategy.GetBuildSteps())
	if err != nil {
		return nil, err
	}

	expectedTaskRun := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: buildRun.Name + "-",
			Namespace:    buildRun.Namespace,
			Labels: map[string]string{
				buildv1alpha1.LabelBuild:              build.Name,
				buildv1alpha1.LabelBuildGeneration:    strconv.FormatInt(build.Generation, 10),
				buildv1alpha1.LabelBuildRun:           buildRun.Name,
				buildv1alpha1.LabelBuildRunGeneration: strconv.FormatInt(buildRun.Generation, 10),
			},
		},
		Spec: v1beta1.TaskRunSpec{
			ServiceAccountName: serviceAccountName,
			TaskSpec:           taskSpec,
			Resources: &v1beta1.TaskRunResources{
				Inputs: []v1beta1.TaskResourceBinding{
					{
						PipelineResourceBinding: v1beta1.PipelineResourceBinding{
							Name: inputSourceResourceName,
							ResourceSpec: &taskv1.PipelineResourceSpec{
								Type: taskv1.PipelineResourceTypeGit,
								Params: []taskv1.ResourceParam{
									{
										Name:  inputGitSourceURL,
										Value: build.Spec.Source.URL,
									},
									{
										Name:  inputGitSourceRevision,
										Value: revision,
									},
								},
							},
						},
					},
				},
				Outputs: []v1beta1.TaskResourceBinding{
					{
						PipelineResourceBinding: v1beta1.PipelineResourceBinding{
							Name: outputImageResourceName,
							ResourceSpec: &taskv1.PipelineResourceSpec{
								Type: taskv1.PipelineResourceTypeImage,
								Params: []taskv1.ResourceParam{
									{
										Name:  outputImageResourceURL,
										Value: ImageURL,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// assign the annotations from the build strategy, filter out those that should not be propagated
	taskRunAnnotations := make(map[string]string)
	for key, value := range strategy.GetAnnotations() {
		if isPropagatableAnnotation(key) {
			taskRunAnnotations[key] = value
		}
	}
	if len(taskRunAnnotations) > 0 {
		expectedTaskRun.Annotations = taskRunAnnotations
	}

	for label, value := range strategy.GetResourceLabels() {
		expectedTaskRun.Labels[label] = value
	}

	expectedTaskRun.Spec.Timeout = effectiveTimeout(build, buildRun)

	var inputParams []v1beta1.Param
	if build.Spec.BuilderImage != nil {
		inputParams = append(inputParams, v1beta1.Param{
			Name: inputParamBuilderImage,
			Value: v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: build.Spec.BuilderImage.ImageURL,
			},
		})
	}
	if build.Spec.Dockerfile != nil {
		inputParams = append(inputParams, v1beta1.Param{
			Name: inputParamDockerfile,
			Value: v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: *build.Spec.Dockerfile,
			},
		})
	}
	if build.Spec.Source.ContextDir != nil {
		inputParams = append(inputParams, v1beta1.Param{
			Name: inputParamContextDir,
			Value: v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: *build.Spec.Source.ContextDir,
			},
		})
	}

	expectedTaskRun.Spec.Params = inputParams
	return expectedTaskRun, nil
}

func effectiveTimeout(build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun) *metav1.Duration {
	if buildRun.Spec.Timeout != nil {
		return buildRun.Spec.Timeout

	} else if build.Spec.Timeout != nil {
		return build.Spec.Timeout
	}

	return nil
}

// isPropagatableAnnotation filters the last-applied-configuration annotation from kubectl because this would break the meaning of this annotation on the target object;
// also, annotations using our own custom resource domains are filtered out because we have no annotations with a semantic for both TaskRun and Pod
func isPropagatableAnnotation(key string) bool {
	return key != "kubectl.kubernetes.io/last-applied-configuration" &&
		!strings.HasPrefix(key, buildv1alpha1.ClusterBuildStrategyDomain+"/") &&
		!strings.HasPrefix(key, buildv1alpha1.BuildStrategyDomain+"/") &&
		!strings.HasPrefix(key, buildv1alpha1.BuildDomain+"/") &&
		!strings.HasPrefix(key, buildv1alpha1.BuildRunDomain+"/")
}

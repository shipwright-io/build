package buildrun

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	taskv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultServiceAccountName  = "default"
	pipelineServiceAccountName = "pipeline"
	inputSourceResourceName    = "source"
	inputGitSourceURL          = "url"
	inputGitSourceRevision     = "revision"
	inputParamBuilderImage     = "BUILDER_IMAGE"
	inputParamDockerfile       = "DOCKERFILE"
	inputParamPathContext      = "PATH_CONTEXT"
	outputImageResourceName    = "image"
	outputImageResourceURL     = "url"
)

// getStringTransformations gets us MANDATORY replacements using
// a poor man's templating mechanism - TODO: Use golang templating
func getStringTransformations(fullText string) string {

	stringTransformations := map[string]string{
		"$(build.output.image)":      "$(outputs.resources.image.url)",
		"$(build.builder.image)":     fmt.Sprintf("$(inputs.params.%s)", inputParamBuilderImage),
		"$(build.dockerfile)":        fmt.Sprintf("$(inputs.params.%s)", inputParamDockerfile),
		"$(build.source.contextDir)": fmt.Sprintf("$(inputs.params.%s)", inputParamPathContext),
	}

	// Run the text through all possible replacements
	for k, v := range stringTransformations {
		fullText = strings.ReplaceAll(fullText, k, v)
	}
	return fullText
}

func GenerateTaskSpec(build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun, buildSteps []buildv1alpha1.BuildStep) (*v1beta1.TaskSpec, error) {

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
				// PATH_CONTEXT comes from the git source specification
				// in the Build object
				Description: "The root of the code",
				Name:        inputParamPathContext,
				Default: &v1beta1.ArrayOrString{
					Type:      v1beta1.ParamTypeString,
					StringVal: ".",
				},
			},
		},
		Steps: []v1beta1.Step{},
	}

	if build.Spec.BuilderImage != nil {
		InputBuilderImage := v1beta1.ParamSpec {
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
				Env:             containerValue.Env,
			},
		}
		if build.Spec.Resources != nil {
			step.Resources = *build.Spec.Resources
			if buildRun.Spec.Resources != nil {
				// Merge the resources from build and buildRun
				mergedResources, err := mergedResources(build, buildRun)
				if err != nil {
					return nil, err
				}
				step.Resources = *mergedResources
			}
		} else {
			if buildRun.Spec.Resources != nil {
				step.Resources = *buildRun.Spec.Resources
			}
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
	return &generatedTaskSpec, nil
}

func GenerateTaskRun(build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun, serviceAccountName string, buildSteps []buildv1alpha1.BuildStep) (*v1beta1.TaskRun, error) {

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

	taskSpec, err := GenerateTaskSpec(build, buildRun, buildSteps)
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

	// assign the timeout
	if buildRun.Spec.Timeout != nil {
		expectedTaskRun.Spec.Timeout = buildRun.Spec.Timeout
	} else if build.Spec.Timeout != nil {
		expectedTaskRun.Spec.Timeout = build.Spec.Timeout
	}

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
			Name: inputParamPathContext,
			Value: v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: *build.Spec.Source.ContextDir,
			},
		})
	}

	expectedTaskRun.Spec.Params = inputParams
	return expectedTaskRun, nil
}

// mergedResources merges the resources from build spec and buildRun spec
func mergedResources(build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun) (*corev1.ResourceRequirements, error) {
	mergedResources := corev1.ResourceRequirements{}
	buildResourceJson, err := json.Marshal(*build.Spec.Resources)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(buildResourceJson, &mergedResources)
	if err != nil {
		return nil, err
	}
	buildRunResourceJson, err := json.Marshal(*buildRun.Spec.Resources)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(buildRunResourceJson, &mergedResources)
	if err != nil {
		return nil, err
	}
	return &mergedResources, nil
}

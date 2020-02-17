package build

import (
	"fmt"
	"strconv"
	"strings"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	taskv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	pipelineServiceAccountName = "pipeline"
	inputParamBuilderImage     = "BUILDER_IMAGE"
	inputParamDockerfile       = "DOCKERFILE"
	inputParamPathContext      = "PATH_CONTEXT"
	outputImageResourceName    = "image"
	outputImageResourceURL     = "url"

	labelBuildGeneration = "build.dev/generation"
	labelBuild           = "build.dev/build"
)

// getStringTransformations gets us MANDATORY replacements using
// a poor man's templating mechanism - TODO: Use golang templating
func getStringTransformations(bs *buildv1alpha1.BuildStrategy, fullText string) string {
	stringTransformations := map[string]string{
		"$(build.outputImage)":  "$(outputs.resources.image.url)",
		"$(build.builderImage)": fmt.Sprintf("$(inputs.params.%s)", inputParamBuilderImage),
		"$(build.dockerfile)":   fmt.Sprintf("$(inputs.params.%s)", inputParamDockerfile),
		"$(build.pathContext)":  fmt.Sprintf("$(inputs.params.%s)", inputParamPathContext),
	}
	// Run the text through all possible replacements
	for k, v := range stringTransformations {
		fullText = strings.ReplaceAll(fullText, k, v)
	}
	return fullText
}

func getCustomTask(buildInstance *buildv1alpha1.Build, buildStrategyInstance *buildv1alpha1.BuildStrategy) *taskv1.Task {

	generatedTask := taskv1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildInstance.Name,
			Namespace: buildInstance.Namespace,
			Labels: map[string]string{
				labelBuild:           buildInstance.Name,
				labelBuildGeneration: strconv.FormatInt(buildInstance.GetGeneration(), 10),
			},
		},
		Spec: taskv1.TaskSpec{
			Inputs: &taskv1.Inputs{
				Params: []taskv1.ParamSpec{
					{
						Description: "Image containing the build tools/logic",
						Name:        inputParamBuilderImage,
						Default: &taskv1.ArrayOrString{
							Type:      taskv1.ParamTypeString,
							StringVal: "docker.io/centos/nodejs-8-centos7", // not really needed.
						},
					},
					{
						Description: "Path to the Dockerfile",
						Name:        inputParamDockerfile,
						Default: &taskv1.ArrayOrString{
							Type:      taskv1.ParamTypeString,
							StringVal: "Dockerfile",
						},
					},
					{
						// PATH_CONTEXT comes from the git source specification
						// in the Build object
						Description: "The root of the code",
						Name:        inputParamPathContext,
						Default: &taskv1.ArrayOrString{
							Type:      taskv1.ParamTypeString,
							StringVal: ".",
						},
					},
				},
				Resources: []taskv1.TaskResource{
					{
						ResourceDeclaration: taskv1.ResourceDeclaration{
							Name: "source",
							Type: taskv1.PipelineResourceTypeGit,
						},
					},
				},
			},
			Outputs: &taskv1.Outputs{
				Resources: []taskv1.TaskResource{
					{
						ResourceDeclaration: taskv1.ResourceDeclaration{
							Name: outputImageResourceName, // mapped from {{ .Build.OutputImage }}
							Type: taskv1.PipelineResourceTypeImage,
						},
					},
				},
			},
			Steps: []v1alpha1.Step{},
		},
	}

	var vols []corev1.Volume

	for _, containerValue := range buildStrategyInstance.Spec.BuildSteps {

		taskCommand := []string{}
		for _, buildStrategyCommandPart := range containerValue.Command {
			taskCommand = append(taskCommand, getStringTransformations(buildStrategyInstance, buildStrategyCommandPart))
		}

		taskArgs := []string{}
		for _, buildStrategyArgPart := range containerValue.Args {
			taskArgs = append(taskArgs, getStringTransformations(buildStrategyInstance, buildStrategyArgPart))
		}

		taskImage := getStringTransformations(buildStrategyInstance, containerValue.Image)

		step := v1alpha2.Step{
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

		generatedTask.Spec.Steps = append(generatedTask.Spec.Steps, step)

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
	generatedTask.Spec.Volumes = vols
	return &generatedTask
}

func applyCredentials(buildInstance *buildv1alpha1.Build, serviceAccount *corev1.ServiceAccount) *corev1.ServiceAccount {
	sourceSecret := buildInstance.Spec.Source.SecretRef
	if sourceSecret == nil {
		return serviceAccount
	}

	isSecretPresent := false
	for _, credentialSecret := range serviceAccount.Secrets {
		if credentialSecret.Name == sourceSecret.Name {
			isSecretPresent = true
			break
		}
	}

	if !isSecretPresent {
		serviceAccount.Secrets = append(serviceAccount.Secrets, corev1.ObjectReference{
			Name: sourceSecret.Name,
		})
	}

	return serviceAccount
}

func getCustomTaskRun(buildInstance *buildv1alpha1.Build, buildStrategyInstance *buildv1alpha1.BuildStrategy) *taskv1.TaskRun {
	expectedTaskRun := &taskv1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildInstance.Name,
			Namespace: buildInstance.Namespace,
			Labels: map[string]string{
				labelBuild: buildInstance.Name,
			},
		},
		Spec: taskv1.TaskRunSpec{
			ServiceAccountName: pipelineServiceAccountName,
			TaskRef: &v1alpha1.TaskRef{
				Name: buildInstance.Name,
			},
			Inputs: taskv1.TaskRunInputs{
				Resources: []taskv1.TaskResourceBinding{
					{
						PipelineResourceBinding: taskv1.PipelineResourceBinding{
							Name: "source",
							ResourceSpec: &taskv1.PipelineResourceSpec{
								Type: taskv1.PipelineResourceTypeGit,
								Params: []taskv1.ResourceParam{
									{
										Name:  "url",
										Value: buildInstance.Spec.Source.URL,
									},
								},
							},
						},
					},
				},
			},
			Outputs: taskv1.TaskRunOutputs{
				Resources: []taskv1.TaskResourceBinding{
					{
						PipelineResourceBinding: taskv1.PipelineResourceBinding{
							Name: outputImageResourceName,
							ResourceSpec: &taskv1.PipelineResourceSpec{
								Type: taskv1.PipelineResourceTypeImage,
								Params: []taskv1.ResourceParam{
									{
										Name:  outputImageResourceURL,
										Value: buildInstance.Spec.OutputImage,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	inputParams := []taskv1.Param{}
	if buildInstance.Spec.BuilderImage != nil {
		inputParams = append(inputParams, taskv1.Param{

			Name: inputParamBuilderImage,
			Value: taskv1.ArrayOrString{
				Type:      taskv1.ParamTypeString,
				StringVal: *buildInstance.Spec.BuilderImage,
			},
		})
	}
	if buildInstance.Spec.Dockerfile != nil {
		inputParams = append(inputParams, taskv1.Param{
			Name: inputParamDockerfile,
			Value: taskv1.ArrayOrString{
				Type:      taskv1.ParamTypeString,
				StringVal: *buildInstance.Spec.Dockerfile,
			},
		})
	}
	if buildInstance.Spec.PathContext != nil {
		inputParams = append(inputParams, taskv1.Param{
			Name: inputParamPathContext,
			Value: taskv1.ArrayOrString{
				Type:      taskv1.ParamTypeString,
				StringVal: *buildInstance.Spec.PathContext,
			},
		})
	}

	expectedTaskRun.Spec.Inputs.Params = inputParams
	return expectedTaskRun
}

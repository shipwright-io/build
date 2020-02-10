package build

import (
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
	inputParamPathContext      = "PATH_CONTEXT"
	outputImageResourceName    = "image"
	outputImageResourceURL     = "url"
)

// getStringTransformations gets us MANDATORY replacements using
// a poor man's templating mechanism - TODO: Use golang templating
func getStringTransformations(bs *buildv1alpha1.BuildStrategy, fullText string) string {
	stringTransformations := map[string]string{
		"$(build.outputImage)":  "$(outputs.resources.image.url)",
		"$(build.builderImage)": "$(inputs.params.BUILDER_IMAGE)",
	}
	// Run the text through all possible replacements
	for k, v := range stringTransformations {
		fullText = strings.ReplaceAll(fullText, k, v)
	}
	return fullText
}

func getCustomTask(instance *buildv1alpha1.Build, bs *buildv1alpha1.BuildStrategy) *taskv1.Task {
	generatedTask := taskv1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"build.dev/build": instance.Name,
			},
		},
		Spec: taskv1.TaskSpec{
			Inputs: &taskv1.Inputs{
				// Every build MUST have a Builder Image and a Context.
				Params: []taskv1.ParamSpec{
					{
						Description: "",
						Name:        inputParamBuilderImage,
					},
					{
						// PATH_CONTEXT comes from the git source specification
						// in the Build object
						Description: "",
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

	for _, containerValue := range bs.Spec.BuildSteps {

		taskCommand := []string{}
		for _, buildStrategyCommandPart := range containerValue.Command {
			taskCommand = append(taskCommand, getStringTransformations(bs, buildStrategyCommandPart))
		}

		taskArgs := []string{}
		for _, buildStrategyArgPart := range containerValue.Args {
			taskArgs = append(taskCommand, getStringTransformations(bs, buildStrategyArgPart))
		}

		taskImage := getStringTransformations(bs, containerValue.Image)

		step := v1alpha2.Step{
			Container: corev1.Container{
				Image:           taskImage,
				Name:            containerValue.Name,
				VolumeMounts:    containerValue.VolumeMounts,
				Command:         taskCommand,
				Args:            taskArgs,
				SecurityContext: containerValue.SecurityContext,
				WorkingDir:      containerValue.WorkingDir,
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

func getCustomTaskRun(instance *buildv1alpha1.Build, bs *buildv1alpha1.BuildStrategy) *taskv1.TaskRun {
	expectedTaskRun := &taskv1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"build.dev/build": instance.Name,
			},
		},
		Spec: taskv1.TaskRunSpec{
			ServiceAccountName: pipelineServiceAccountName,
			TaskRef: &v1alpha1.TaskRef{
				Name: instance.Name,
			},
			Inputs: taskv1.TaskRunInputs{
				Params: []taskv1.Param{
					{
						Name: "BUILDER_IMAGE",
						Value: taskv1.ArrayOrString{
							Type:      taskv1.ParamTypeString,
							StringVal: *instance.Spec.BuilderImage,
						},
					},
				},
				Resources: []taskv1.TaskResourceBinding{
					{
						PipelineResourceBinding: taskv1.PipelineResourceBinding{
							Name: "source",
							ResourceSpec: &taskv1.PipelineResourceSpec{
								Type: taskv1.PipelineResourceTypeGit,
								Params: []taskv1.ResourceParam{
									{
										Name:  "url",
										Value: instance.Spec.Source.URL,
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
										Value: instance.Spec.OutputImage,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return expectedTaskRun
}

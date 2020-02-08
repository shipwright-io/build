package build

import (
	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	taskv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getBuildpacksV3TaskRun(instance *buildv1alpha1.Build) *taskv1.TaskRun {
	expectedTaskRun := &taskv1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{Name: instance.Name, Namespace: instance.Namespace},
		Spec: taskv1.TaskRunSpec{
			ServiceAccountName: "pipeline",
			TaskRef: &v1alpha1.TaskRef{
				Name: instance.Name,
			},
			PodTemplate: &taskv1.PodTemplate{
				Volumes: []corev1.Volume{
					{
						Name: "my-cache",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "my-volume-claim",
							},
						},
					},
				},
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
					{
						Name: "CACHE",
						Value: taskv1.ArrayOrString{
							Type:      taskv1.ParamTypeString,
							StringVal: "my-cache",
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
							Name: "image",
							ResourceSpec: &taskv1.PipelineResourceSpec{
								Type: taskv1.PipelineResourceTypeImage,
								Params: []taskv1.ResourceParam{
									{
										Name:  "url",
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

func getBuildpacksV3Task(instance *buildv1alpha1.Build) *taskv1.Task {
	truePr := true
	expectedTask := &taskv1.Task{
		ObjectMeta: metav1.ObjectMeta{Name: instance.Name, Namespace: instance.Namespace},
		Spec: taskv1.TaskSpec{
			Steps: []taskv1.Step{
				{
					Container: corev1.Container{
						Name:  "generate",
						Image: "quay.io/openshift-pipeline/s2i:nightly",
						Command: []string{
							"/usr/local/bin/s2i",
							"--loglevel=$(inputs.params.LOGLEVEL)",
							"build",
							"$(inputs.params.PATH_CONTEXT)",
							"$(inputs.params.BUILDER_IMAGE)",
							"--as-dockerfile",
							"/gen-source/Dockerfile.gen",
						},
						//Args:       []string{"--my-other-arg=$(inputs.resources.workspace.url)"},
						WorkingDir: "/workspace/source",
						VolumeMounts: []corev1.VolumeMount{
							{
								MountPath: "/gen-source",
								Name:      "gen-source",
							},
						},
					},
				},
				{
					Container: corev1.Container{
						Name:  "build",
						Image: "quay.io/buildah/stable",
						Command: []string{
							"buildah",
							"bud",
							"--tls-verify=$(inputs.params.TLS_VERIFY)",
							"--layers",
							"-f",
							"/gen-source/Dockerfile.gen",
							"-t",
							"$(outputs.resources.image.url)",
							".",
						},
						//Args:       []string{"--my-other-arg=$(inputs.resources.workspace.url)"},
						WorkingDir: "/gen-source",
						VolumeMounts: []corev1.VolumeMount{
							{
								MountPath: "/var/lib/containers",
								Name:      "varlibcontainers",
							},
							{
								MountPath: "/gen-source",
								Name:      "gen-source",
							},
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &truePr,
						},
					},
				},
				{
					Container: corev1.Container{
						Name:  "push",
						Image: "quay.io/buildah/stable",
						Command: []string{
							"buildah",
							"push",
							"--tls-verify=$(inputs.params.TLS_VERIFY)",
							"$(outputs.resources.image.url)",
							"docker://$(outputs.resources.image.url)",
						},
						//Args:       []string{"--my-other-arg=$(inputs.resources.workspace.url)"},
						//WorkingDir: "/workspace/source",
						VolumeMounts: []corev1.VolumeMount{
							{
								MountPath: "/var/lib/containers",
								Name:      "varlibcontainers",
							},
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &truePr,
						},
					},
				},
			},
			Inputs: &taskv1.Inputs{
				Params: []taskv1.ParamSpec{
					{
						Description: "",
						Name:        "BUILDER_IMAGE",
					},
					{
						Description: "",
						Name:        "PATH_CONTEXT",
						Default: &taskv1.ArrayOrString{
							Type:      taskv1.ParamTypeString,
							StringVal: ".",
						},
					},
					{
						Description: "",
						Name:        "TLS_VERIFY",
						Default: &taskv1.ArrayOrString{
							Type:      taskv1.ParamTypeString,
							StringVal: "false",
						},
					},
					{
						Description: "",
						Name:        "LOGLEVEL",
						Default: &taskv1.ArrayOrString{
							Type:      taskv1.ParamTypeString,
							StringVal: "0",
						},
					},
				},
				Resources: []taskv1.TaskResource{
					{
						ResourceDeclaration: taskv1.ResourceDeclaration{
							Name: "source",
							Type: taskv1.PipelineResourceTypeGit,
							//TargetPath: "/foo/bar",
						},
					},
				},
			},
			Outputs: &taskv1.Outputs{
				Resources: []taskv1.TaskResource{
					{
						ResourceDeclaration: taskv1.ResourceDeclaration{
							Name: "image",
							Type: taskv1.PipelineResourceTypeImage,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "varlibcontainers",
				},
				{
					Name: "gen-source",
				},
			},
		},
	}
	return expectedTask
}

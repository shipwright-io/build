package build

import (
	"fmt"
	"reflect"
	"testing"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/stretchr/testify/assert"
	taskv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	buildah = "buildah"
	url     = "https://github.com/sbose78/taxi"
)

func TestGenerateTask(t *testing.T) {

	dockerfile := "Dockerfile"
	builderImage := buildv1alpha1.Image{
		ImageURL: "quay.io/builder/image",
	}
	outputPath := "image-registry.openshift-image-registry.svc:5000/example/taxi-app"
	truePtr := true

	type args struct {
		buildInstance         *buildv1alpha1.Build
		buildStrategyInstance *buildv1alpha1.BuildStrategy
	}
	tests := []struct {
		name string
		args args
		want *taskv1.Task
	}{
		{
			"task generation",
			args{
				buildInstance: &buildv1alpha1.Build{
					ObjectMeta: metav1.ObjectMeta{Name: buildah},
					Spec: buildv1alpha1.BuildSpec{
						Source: buildv1alpha1.GitSource{
							URL: url,
						},
						StrategyRef: metav1.ObjectMeta{
							Name: buildah,
						},
						Dockerfile:   &dockerfile,
						BuilderImage: &builderImage,
						Output: buildv1alpha1.Image{
							ImageURL: outputPath,
						},
					},
				},

				buildStrategyInstance: &buildv1alpha1.BuildStrategy{
					ObjectMeta: metav1.ObjectMeta{Name: buildah},
					Spec: buildv1alpha1.BuildStrategySpec{
						BuildSteps: []buildv1alpha1.BuildStep{
							buildv1alpha1.BuildStep{
								Container: corev1.Container{
									Name:       "build",
									Image:      "$(build.builder.image)",
									WorkingDir: "/workspace/source",
									Command: []string{
										"buildah", "bud", "--tls-verify=false", "--layers", "-f", "$(build.dockerfile)", "-t", "$(build.output.image)", "$(build.pathContext)",
									},
									Args: []string{
										"buildah", "bud", "--tls-verify=false", "--layers", "-f", "$(build.dockerfile)", "-t", "$(build.output.image)", "$(build.pathContext)",
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "varlibcontainers",
											MountPath: "/var/lib/containers",
										},
									},
									SecurityContext: &corev1.SecurityContext{
										Privileged: &truePtr,
									},
								},
							},
							buildv1alpha1.BuildStep{
								Container: corev1.Container{
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "varlibcontainers",
											MountPath: "/var/lib/containers",
										},
										{
											Name:      "something-else",
											MountPath: "/var/lib/containers",
										},
									},
								},
							},
						},
					},
				},
			},
			&taskv1.Task{}, // not using it for now
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCustomTask(tt.args.buildInstance, tt.args.buildStrategyInstance)
			expectedCommandOrArg := []string{
				"buildah", "bud", "--tls-verify=false", "--layers", "-f", fmt.Sprintf("$(inputs.params.%s)", inputParamDockerfile), "-t", "$(outputs.resources.image.url)", fmt.Sprintf("$(inputs.params.%s)", inputParamPathContext),
			}

			// ensure IMAGE is replaced by builder image when needed.
			assert.Equal(t, fmt.Sprintf("$(inputs.params.%s)", inputParamBuilderImage), got.Spec.Steps[0].Container.Image)

			// ensure command replacements happen when needed
			assert.True(t, reflect.DeepEqual(got.Spec.Steps[0].Container.Command, expectedCommandOrArg))

			// ensure arg replacements happen when needed.
			assert.True(t, reflect.DeepEqual(expectedCommandOrArg, got.Spec.Steps[0].Container.Args))

			// Ensure top level volumes are populated.
			assert.Equal(t, 2, len(got.Spec.Volumes))
		})
	}
}

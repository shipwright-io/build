package buildrun

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/stretchr/testify/assert"
	taskv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	buildah = "buildah"
	buildpacks = "buildpacks-v3"
	url     = "https://github.com/sbose78/taxi"
)

func TestGenerateTaskSpec(t *testing.T) {

	dockerfile := "Dockerfile"
	builderImage := buildv1alpha1.Image{
		ImageURL: "quay.io/builder/image",
	}
	buildStrategy := buildv1alpha1.ClusterBuildStrategyKind
	truePtr := true

	type args struct {
		build         *buildv1alpha1.Build
		buildStrategy *buildv1alpha1.BuildStrategy
	}
	tests := []struct {
		name string
		args args
		want *taskv1.TaskSpec
	}{
		{
			"taskSpec generation",
			args{
				build: &buildv1alpha1.Build{
					ObjectMeta: metav1.ObjectMeta{Name: buildah},
					Spec: buildv1alpha1.BuildSpec{
						Source: buildv1alpha1.GitSource{
							URL: url,
						},
						StrategyRef: &buildv1alpha1.StrategyRef{
							Name: buildah,
							Kind: &buildStrategy,
						},
						Dockerfile:   &dockerfile,
						BuilderImage: &builderImage,
					},
				},

				buildStrategy: &buildv1alpha1.BuildStrategy{
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
			&taskv1.TaskSpec{}, // not using it for now
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateTaskSpec(tt.args.build, tt.args.buildStrategy.Spec.BuildSteps)
			expectedCommandOrArg := []string{
				"buildah", "bud", "--tls-verify=false", "--layers", "-f", fmt.Sprintf("$(inputs.params.%s)", inputParamDockerfile), "-t", "$(outputs.resources.image.url)", fmt.Sprintf("$(inputs.params.%s)", inputParamPathContext),
			}

			// ensure IMAGE is replaced by builder image when needed.
			assert.Equal(t, fmt.Sprintf("$(inputs.params.%s)", inputParamBuilderImage), got.Steps[0].Container.Image)

			// ensure command replacements happen when needed
			assert.True(t, reflect.DeepEqual(got.Steps[0].Container.Command, expectedCommandOrArg))

			// ensure arg replacements happen when needed.
			assert.True(t, reflect.DeepEqual(expectedCommandOrArg, got.Steps[0].Container.Args))

			// Ensure top level volumes are populated.
			assert.Equal(t, 2, len(got.Volumes))
		})
	}
}

func TestGenerateTaskRun(t *testing.T) {

	namespace := "build-test"
	dockerfile := "Dockerfile"
	ContextDir := "src"
	builderImage := buildv1alpha1.Image{
		ImageURL: "heroku/buildpacks:18",
	}
	clustertBuildStrategy := buildv1alpha1.ClusterBuildStrategyKind
	outputPath := "image-registry.openshift-image-registry.svc:5000/example/buildpacks-app"
	serviceAccountName := buildpacks + "-serviceaccount"

	type args struct {
		build         *buildv1alpha1.Build
		buildRun      *buildv1alpha1.BuildRun
	}
	tests := []struct {
		name string
		args args
		want *taskv1.TaskRun
	}{
		{
			"taskrun generation",
			args{
				build: &buildv1alpha1.Build{
					ObjectMeta: metav1.ObjectMeta{
						Name: buildpacks,
						Namespace: namespace,
					},
					Spec: buildv1alpha1.BuildSpec{
						Source: buildv1alpha1.GitSource{
							URL: url,
							ContextDir: &ContextDir,
						},
						StrategyRef: &buildv1alpha1.StrategyRef{
							Name: buildpacks,
							Kind: &clustertBuildStrategy,
						},
						Dockerfile:   &dockerfile,
						BuilderImage: &builderImage,
						Output: buildv1alpha1.Image{
							ImageURL: outputPath,
						},
					},
				},
				buildRun: &buildv1alpha1.BuildRun{
					ObjectMeta: metav1.ObjectMeta{
						Name: buildpacks + "-run",
						Namespace: namespace,
					},
					Spec: buildv1alpha1.BuildRunSpec{
						BuildRef: &buildv1alpha1.BuildRef{
							Name:       buildpacks,
						},
						ServiceAccount: &serviceAccountName,
					},
				},
			},
			&taskv1.TaskRun{}, // not using it for now
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateTaskRun(tt.args.build, tt.args.buildRun, serviceAccountName, nil)

			// ensure generated TaskRun's basic information are correct
			assert.True(t, reflect.DeepEqual(true, strings.Contains(got.GenerateName, tt.args.buildRun.Name + "-")))
			assert.True(t, reflect.DeepEqual(got.Namespace, namespace))
			assert.True(t, reflect.DeepEqual(got.Spec.ServiceAccountName, buildpacks + "-serviceaccount"))
			assert.True(t, reflect.DeepEqual(got.Labels[buildv1alpha1.LabelBuild], tt.args.build.Name))
			assert.True(t, reflect.DeepEqual(got.Labels[buildv1alpha1.LabelBuildRun], tt.args.buildRun.Name))

			// ensure generated TaskRun's input and output resources are correct
			inputResources := got.Spec.Inputs.Resources
			for _, inputResource := range inputResources {
				if inputResource.Name == inputImageResourceName {
					assert.True(t, reflect.DeepEqual(inputResource.ResourceSpec.Type, taskv1.PipelineResourceTypeGit))
					params := inputResource.ResourceSpec.Params
					for _, param := range params {
						if param.Name == inputImageResourceURL {
							assert.True(t, reflect.DeepEqual(param.Value, url))
						}
					}

				}
			}
			outputResources := got.Spec.Outputs.Resources
			for _, outputResource := range outputResources {
				if outputResource.Name == outputImageResourceName {
					assert.True(t, reflect.DeepEqual(outputResource.ResourceSpec.Type, taskv1.PipelineResourceTypeImage))
					params := outputResource.ResourceSpec.Params
					for _, param := range params {
						if param.Name == outputImageResourceURL {
							assert.True(t, reflect.DeepEqual(param.Value, outputPath))
						}
					}

				}
			}

			// ensure generated TaskRun's spec special input params are correct
			params := got.Spec.Inputs.Params
			for _, param := range params {
				if param.Name == inputParamBuilderImage {
					assert.True(t, reflect.DeepEqual(param.Value.StringVal, builderImage.ImageURL))
				}
				if param.Name == inputParamDockerfile {
					assert.True(t, reflect.DeepEqual(param.Value.StringVal, dockerfile))
				}
				if param.Name == inputParamPathContext {
					assert.True(t, reflect.DeepEqual(param.Value.StringVal, ContextDir))
				}
			}
		})
	}
}

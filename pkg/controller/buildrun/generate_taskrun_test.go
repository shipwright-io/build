package buildrun_test

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	buildrunCtl "github.com/redhat-developer/build/pkg/controller/buildrun"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("GenerateTaskrun", func() {

	var (
		build                                *buildv1alpha1.Build
		buildRun                             *buildv1alpha1.BuildRun
		buildStrategy                        *buildv1alpha1.BuildStrategy
		builderImage                         *buildv1alpha1.Image
		clusterBuildStrategy                 buildv1alpha1.BuildStrategyKind
		dockerfile, buildah, buildpacks, url string
	)

	BeforeEach(func() {
		buildah = "buildah"
		buildpacks = "buildpacks-v3"
		url = "https://github.com/sbose78/taxi"
		dockerfile = "Dockerfile"
		clusterBuildStrategy = buildv1alpha1.ClusterBuildStrategyKind
	})

	Describe("Generate the TaskSpec", func() {
		var (
			truePtr               bool
			expectedCommandOrArg  []string
			expectedResourceOrArg corev1.ResourceRequirements
			got                   *v1beta1.TaskSpec
			err                   error
		)
		BeforeEach(func() {
			builderImage = &buildv1alpha1.Image{
				ImageURL: "quay.io/builder/image",
			}
			truePtr = true
		})

		Context("when the task spec is generated", func() {
			BeforeEach(func() {
				build = &buildv1alpha1.Build{
					ObjectMeta: metav1.ObjectMeta{Name: buildah},
					Spec: buildv1alpha1.BuildSpec{
						Source: buildv1alpha1.GitSource{
							URL: url,
						},
						StrategyRef: &buildv1alpha1.StrategyRef{
							Name: buildah,
							Kind: &clusterBuildStrategy,
						},
						Dockerfile:   &dockerfile,
						BuilderImage: builderImage,
						Resources: &corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
						},
					},
				}

				buildRun = &buildv1alpha1.BuildRun{
					ObjectMeta: metav1.ObjectMeta{
						Name: buildah + "-run",
					},
					Spec: buildv1alpha1.BuildRunSpec{
						BuildRef: &buildv1alpha1.BuildRef{
							Name: buildah,
						},
					},
				}

				buildStrategy = &buildv1alpha1.BuildStrategy{
					ObjectMeta: metav1.ObjectMeta{Name: buildah},
					Spec: buildv1alpha1.BuildStrategySpec{
						BuildSteps: []buildv1alpha1.BuildStep{
							{
								Container: corev1.Container{
									Name:       "build",
									Image:      "$(build.builder.image)",
									WorkingDir: "/workspace/source",
									Command: []string{
										"buildah", "bud", "--tls-verify=false", "--layers", "-f", "$(build.dockerfile)", "-t", "$(build.output.image)", "$(build.source.contextDir)",
									},
									Args: []string{
										"buildah", "bud", "--tls-verify=false", "--layers", "-f", "$(build.dockerfile)", "-t", "$(build.output.image)", "$(build.source.contextDir)",
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
							{
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
				}

				expectedCommandOrArg = []string{
					"buildah", "bud", "--tls-verify=false", "--layers", "-f", fmt.Sprintf("$(inputs.params.%s)", "DOCKERFILE"), "-t", "$(outputs.resources.image.url)", fmt.Sprintf("$(inputs.params.%s)", "PATH_CONTEXT"),
				}

				expectedResourceOrArg = corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
				}
			})

			JustBeforeEach(func() {
				got, err = buildrunCtl.GenerateTaskSpec(build, buildRun, buildStrategy.Spec.BuildSteps)
				Expect(err).To(BeNil())
			})

			It("should ensure IMAGE is replaced by builder image when needed.", func() {
				Expect(got.Steps[0].Container.Image).To(Equal(fmt.Sprintf("$(inputs.params.%s)", "BUILDER_IMAGE")))
			})

			It("should ensure command replacements happen when needed", func() {
				Expect(got.Steps[0].Container.Command).To(Equal(expectedCommandOrArg))
			})

			It("should ensure resource replacements happen when needed", func() {
				Expect(got.Steps[0].Container.Resources).To(Equal(expectedResourceOrArg))
			})

			It("should ensure arg replacements happen when needed", func() {
				Expect(got.Steps[0].Container.Args).To(Equal(expectedCommandOrArg))
			})

			It("should ensure top level volumes are populated", func() {
				Expect(len(got.Volumes)).To(Equal(2))
			})
		})
	})

	Describe("Generate the TaskRun", func() {
		var (
			namespace, contextDir, revision, outputPath, serviceAccountName string
			got                                                             *v1beta1.TaskRun
			err                                                             error
		)
		BeforeEach(func() {
			namespace = "build-test"
			contextDir = "src"
			revision = "master"
			builderImage = &buildv1alpha1.Image{
				ImageURL: "heroku/buildpacks:18",
			}
			outputPath = "image-registry.openshift-image-registry.svc:5000/example/buildpacks-app"
			serviceAccountName = buildpacks + "-serviceaccount"
		})

		Context("when the taskrun is generated by default", func() {
			BeforeEach(func() {
				build = &buildv1alpha1.Build{
					ObjectMeta: metav1.ObjectMeta{
						Name:      buildah,
						Namespace: namespace,
					},
					Spec: buildv1alpha1.BuildSpec{
						Source: buildv1alpha1.GitSource{
							URL: url,
						},
						StrategyRef: &buildv1alpha1.StrategyRef{
							Name: buildah,
						},
						Output: buildv1alpha1.Image{
							ImageURL: outputPath,
						},
					},
				}
				buildRun = &buildv1alpha1.BuildRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      buildah + "-run",
						Namespace: namespace,
					},
					Spec: buildv1alpha1.BuildRunSpec{
						BuildRef: &buildv1alpha1.BuildRef{
							Name: buildah,
						},
						Resources: &corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("2Gi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("2Gi"),
							},
						},
						ServiceAccount: &buildv1alpha1.ServiceAccount{
							Name: &serviceAccountName,
						},
					},
				}
				buildStrategy = &buildv1alpha1.BuildStrategy{
					ObjectMeta: metav1.ObjectMeta{Name: buildah},
					Spec: buildv1alpha1.BuildStrategySpec{
						BuildSteps: []buildv1alpha1.BuildStep{
							{
								Container: corev1.Container{
									Name:       "build",
									Image:      "$(build.builder.image)",
									WorkingDir: "/workspace/source",
									Command: []string{
										"buildah", "bud", "--tls-verify=false", "--layers", "-f", "$(build.dockerfile)", "-t", "$(build.output.image)", "$(build.pathContext)",
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "varlibcontainers",
											MountPath: "/var/lib/containers",
										},
									},
								},
							},
						},
					},
				}
			})

			JustBeforeEach(func() {
				got, err = buildrunCtl.GenerateTaskRun(build, buildRun, serviceAccountName, buildStrategy.Spec.BuildSteps)
				Expect(err).To(BeNil())
			})

			It("should ensure generated TaskRun's basic information are correct", func() {
				Expect(strings.Contains(got.GenerateName, buildRun.Name+"-")).To(Equal(true))
				Expect(got.Namespace).To(Equal(namespace))
				Expect(got.Spec.ServiceAccountName).To(Equal(buildpacks + "-serviceaccount"))
				Expect(got.Labels[buildv1alpha1.LabelBuild]).To(Equal(build.Name))
				Expect(got.Labels[buildv1alpha1.LabelBuildRun]).To(Equal(buildRun.Name))
			})

			It("should ensure generated TaskRun's input and output resources are correct", func() {
				inputResources := got.Spec.Resources.Inputs
				for _, inputResource := range inputResources {
					Expect(inputResource.ResourceSpec.Type).To(Equal(v1beta1.PipelineResourceTypeGit))
					params := inputResource.ResourceSpec.Params
					for _, param := range params {
						if param.Name == "url" {
							Expect(param.Value).To(Equal(url))
						}
						if param.Name == "revision" {
							Expect(param.Value).To(Equal(revision))
						}
					}
				}

				outputResources := got.Spec.Resources.Outputs
				for _, outputResource := range outputResources {
					Expect(outputResource.ResourceSpec.Type).To(Equal(v1beta1.PipelineResourceTypeImage))
					params := outputResource.ResourceSpec.Params
					for _, param := range params {
						if param.Name == "url" {
							Expect(param.Value).To(Equal(outputPath))
						}
					}
				}
			})

			It("should ensure resource replacements happen when needed", func() {
				expectedResourceOrArg := corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				}
				Expect(got.Spec.TaskSpec.Steps[0].Resources).To(Equal(expectedResourceOrArg))
			})
		})

		Context("when the taskrun is generated by special settings", func() {
			BeforeEach(func() {
				build = &buildv1alpha1.Build{
					ObjectMeta: metav1.ObjectMeta{
						Name:      buildpacks,
						Namespace: namespace,
					},
					Spec: buildv1alpha1.BuildSpec{
						Source: buildv1alpha1.GitSource{
							URL:        url,
							Revision:   &revision,
							ContextDir: &contextDir,
						},
						StrategyRef: &buildv1alpha1.StrategyRef{
							Name: buildpacks,
							Kind: &clusterBuildStrategy,
						},
						Dockerfile:   &dockerfile,
						BuilderImage: builderImage,
						Output: buildv1alpha1.Image{
							ImageURL: outputPath,
						},
						Resources: &corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
						},
					},
				}
				buildRun = &buildv1alpha1.BuildRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      buildpacks + "-run",
						Namespace: namespace,
					},
					Spec: buildv1alpha1.BuildRunSpec{
						BuildRef: &buildv1alpha1.BuildRef{
							Name: buildpacks,
						},
						Resources: &corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("2Gi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("2Gi"),
							},
						},
						ServiceAccount: &buildv1alpha1.ServiceAccount{
							Name:     &serviceAccountName,
							Generate: false,
						},
					},
				}
				buildStrategy = &buildv1alpha1.BuildStrategy{
					ObjectMeta: metav1.ObjectMeta{Name: buildpacks},
					Spec: buildv1alpha1.BuildStrategySpec{
						BuildSteps: []buildv1alpha1.BuildStep{
							{
								Container: corev1.Container{
									Name:       "build",
									Image:      "$(build.builder.image)",
									WorkingDir: "/workspace/source",
									Command: []string{
										"/cnb/lifecycle/builder", "-app", "/workspace/source", "-layers", "/layers", "-group", "/layers/group.toml", "plan", "/layers/plan.toml",
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "varlibcontainers",
											MountPath: "/var/lib/containers",
										},
									},
								},
							},
						},
					},
				}
			})

			JustBeforeEach(func() {
				got, err = buildrunCtl.GenerateTaskRun(build, buildRun, serviceAccountName, buildStrategy.Spec.BuildSteps)
				Expect(err).To(BeNil())
			})

			It("should ensure generated TaskRun's basic information are correct", func() {
				Expect(strings.Contains(got.GenerateName, buildRun.Name+"-")).To(Equal(true))
				Expect(got.Namespace).To(Equal(namespace))
				Expect(got.Spec.ServiceAccountName).To(Equal(buildpacks + "-serviceaccount"))
				Expect(got.Labels[buildv1alpha1.LabelBuild]).To(Equal(build.Name))
				Expect(got.Labels[buildv1alpha1.LabelBuildRun]).To(Equal(buildRun.Name))
			})

			It("should ensure generated TaskRun's spec special input params are correct", func() {
				params := got.Spec.Params
				for _, param := range params {
					if param.Name == "BUILDER_IMAGE" {
						Expect(param.Value.StringVal).To(Equal(builderImage.ImageURL))
					}
					if param.Name == "DOCKERFILE" {
						Expect(param.Value.StringVal).To(Equal(dockerfile))
					}
					if param.Name == "PATH_CONTEXT" {
						Expect(param.Value.StringVal).To(Equal(contextDir))
					}
				}
			})

			It("should ensure resource replacements happen when needed", func() {
				expectedResourceOrArg := corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				}
				Expect(got.Spec.TaskSpec.Steps[0].Resources).To(Equal(expectedResourceOrArg))
			})
		})
	})
})

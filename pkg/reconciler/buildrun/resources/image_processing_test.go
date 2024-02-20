// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	utils "github.com/shipwright-io/build/test/utils/v1beta1"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

var _ = Describe("Image Processing overrides", func() {

	config := config.NewDefaultConfig()
	var processedTaskRun *pipelineapi.TaskRun

	Context("for a TaskRun that does not reference the output directory", func() {
		taskRun := &pipelineapi.TaskRun{
			Spec: pipelineapi.TaskRunSpec{
				TaskSpec: &pipelineapi.TaskSpec{
					Steps: []pipelineapi.Step{
						{
							Name: "test-step",
						},
					},
				},
			},
		}

		Context("for a build without labels and annotation in the output", func() {
			BeforeEach(func() {
				processedTaskRun = taskRun.DeepCopy()
				resources.SetupImageProcessing(processedTaskRun, config, buildv1beta1.Image{
					Image: "some-registry/some-namespace/some-image",
				}, buildv1beta1.Image{})
			})

			It("does not add the image-processing step", func() {
				Expect(processedTaskRun.Spec.TaskSpec.Steps).To(HaveLen(1))
				Expect(processedTaskRun.Spec.TaskSpec.Steps).ToNot(utils.ContainNamedElement("image-processing"))
			})
		})

		Context("for a build with a label in the output", func() {
			BeforeEach(func() {
				processedTaskRun = taskRun.DeepCopy()
				resources.SetupImageProcessing(processedTaskRun, config, buildv1beta1.Image{
					Image: "some-registry/some-namespace/some-image",
					Labels: map[string]string{
						"aKey": "aLabel",
					},
				}, buildv1beta1.Image{})
			})

			It("adds the image-processing step", func() {
				Expect(processedTaskRun.Spec.TaskSpec.Steps).To(HaveLen(2))
				Expect(processedTaskRun.Spec.TaskSpec.Steps[1].Name).To(Equal("image-processing"))
				Expect(processedTaskRun.Spec.TaskSpec.Steps[1].Image).To(Equal(config.ImageProcessingContainerTemplate.Image))
				Expect(processedTaskRun.Spec.TaskSpec.Steps[1].Command).To(Equal(config.ImageProcessingContainerTemplate.Command))
				Expect(processedTaskRun.Spec.TaskSpec.Steps[1].Args).To(Equal([]string{
					"--label",
					"aKey=aLabel",
					"--image",
					"$(params.shp-output-image)",
					"--insecure=$(params.shp-output-insecure)",
					"--result-file-image-digest",
					"$(results.shp-image-digest.path)",
					"--result-file-image-size",
					"$(results.shp-image-size.path)",
				}))
				Expect(processedTaskRun.Spec.TaskSpec.Steps[1].VolumeMounts).ToNot(utils.ContainNamedElement("shp-output-directory"))
			})
		})
	})

	Context("for a TaskRun that references the output directory", func() {

		taskRun := &pipelineapi.TaskRun{
			Spec: pipelineapi.TaskRunSpec{
				TaskSpec: &pipelineapi.TaskSpec{
					Steps: []pipelineapi.Step{
						{
							Name: "test-step",
							Args: []string{
								"$(params.shp-output-directory)",
							},
						},
					},
				},
			},
		}

		Context("for a build with an output without a secret", func() {

			Context("for a build with label and annotation in the output", func() {
				BeforeEach(func() {
					processedTaskRun = taskRun.DeepCopy()
					resources.SetupImageProcessing(processedTaskRun, config, buildv1beta1.Image{
						Image: "some-registry/some-namespace/some-image",
						Labels: map[string]string{
							"a-label": "a-value",
						},
					}, buildv1beta1.Image{
						Annotations: map[string]string{
							"an-annotation": "some-value",
						},
					})
				})

				It("adds the output-directory parameter", func() {
					Expect(processedTaskRun.Spec.TaskSpec.Params).To(utils.ContainNamedElement("shp-output-directory"))
					Expect(processedTaskRun.Spec.Params).To(utils.ContainNamedElement("shp-output-directory"))
				})

				It("adds a volume for the output directory", func() {
					Expect(processedTaskRun.Spec.TaskSpec.Volumes).To(utils.ContainNamedElement("shp-output-directory"))
				})

				It("adds the image-processing step", func() {
					Expect(processedTaskRun.Spec.TaskSpec.Steps).To(HaveLen(2))
					Expect(processedTaskRun.Spec.TaskSpec.Steps[1].Name).To(Equal("image-processing"))
					Expect(processedTaskRun.Spec.TaskSpec.Steps[1].Image).To(Equal(config.ImageProcessingContainerTemplate.Image))
					Expect(processedTaskRun.Spec.TaskSpec.Steps[1].Command).To(Equal(config.ImageProcessingContainerTemplate.Command))
					Expect(processedTaskRun.Spec.TaskSpec.Steps[1].Args).To(Equal([]string{
						"--push",
						"$(params.shp-output-directory)",
						"--annotation",
						"an-annotation=some-value",
						"--label",
						"a-label=a-value",
						"--image",
						"$(params.shp-output-image)",
						"--insecure=$(params.shp-output-insecure)",
						"--result-file-image-digest",
						"$(results.shp-image-digest.path)",
						"--result-file-image-size",
						"$(results.shp-image-size.path)",
					}))
					Expect(processedTaskRun.Spec.TaskSpec.Steps[1].VolumeMounts).To(utils.ContainNamedElement("shp-output-directory"))
				})
			})

			Context("for a build without labels and annotation in the output", func() {
				BeforeEach(func() {
					processedTaskRun = taskRun.DeepCopy()
					resources.SetupImageProcessing(processedTaskRun, config, buildv1beta1.Image{
						Image: "some-registry/some-namespace/some-image",
					}, buildv1beta1.Image{})
				})

				It("adds the output-directory parameter", func() {
					Expect(processedTaskRun.Spec.TaskSpec.Params).To(utils.ContainNamedElement("shp-output-directory"))
					Expect(processedTaskRun.Spec.Params).To(utils.ContainNamedElement("shp-output-directory"))
				})

				It("adds a volume for the output directory", func() {
					Expect(processedTaskRun.Spec.TaskSpec.Volumes).To(utils.ContainNamedElement("shp-output-directory"))
				})

				It("adds the image-processing step", func() {
					Expect(processedTaskRun.Spec.TaskSpec.Steps).To(HaveLen(2))
					Expect(processedTaskRun.Spec.TaskSpec.Steps[1].Name).To(Equal("image-processing"))
					Expect(processedTaskRun.Spec.TaskSpec.Steps[1].Image).To(Equal(config.ImageProcessingContainerTemplate.Image))
					Expect(processedTaskRun.Spec.TaskSpec.Steps[1].Command).To(Equal(config.ImageProcessingContainerTemplate.Command))
					Expect(processedTaskRun.Spec.TaskSpec.Steps[1].Args).To(Equal([]string{
						"--push",
						"$(params.shp-output-directory)",
						"--image",
						"$(params.shp-output-image)",
						"--insecure=$(params.shp-output-insecure)",
						"--result-file-image-digest",
						"$(results.shp-image-digest.path)",
						"--result-file-image-size",
						"$(results.shp-image-size.path)",
					}))
					Expect(processedTaskRun.Spec.TaskSpec.Steps[1].VolumeMounts).To(utils.ContainNamedElement("shp-output-directory"))
				})
			})
		})

		Context("for a build with an output with a secret", func() {
			BeforeEach(func() {
				processedTaskRun = taskRun.DeepCopy()
				someSecret := "some-secret"
				resources.SetupImageProcessing(processedTaskRun, config, buildv1beta1.Image{
					Image:      "some-registry/some-namespace/some-image",
					PushSecret: &someSecret,
				}, buildv1beta1.Image{})
			})

			It("adds the output-directory parameter", func() {
				Expect(processedTaskRun.Spec.TaskSpec.Params).To(utils.ContainNamedElement("shp-output-directory"))
				Expect(processedTaskRun.Spec.Params).To(utils.ContainNamedElement("shp-output-directory"))
			})

			It("adds a volume for the output directory", func() {
				Expect(processedTaskRun.Spec.TaskSpec.Volumes).To(utils.ContainNamedElement("shp-output-directory"))
			})

			It("adds a value for the output secret", func() {
				Expect(processedTaskRun.Spec.TaskSpec.Volumes).To(utils.ContainNamedElement("shp-some-secret"))
			})

			It("adds the image-processing step", func() {
				Expect(processedTaskRun.Spec.TaskSpec.Steps).To(HaveLen(2))
				Expect(processedTaskRun.Spec.TaskSpec.Steps[1].Name).To(Equal("image-processing"))
				Expect(processedTaskRun.Spec.TaskSpec.Steps[1].Image).To(Equal(config.ImageProcessingContainerTemplate.Image))
				Expect(processedTaskRun.Spec.TaskSpec.Steps[1].Command).To(Equal(config.ImageProcessingContainerTemplate.Command))
				Expect(processedTaskRun.Spec.TaskSpec.Steps[1].Args).To(Equal([]string{
					"--push",
					"$(params.shp-output-directory)",
					"--image",
					"$(params.shp-output-image)",
					"--insecure=$(params.shp-output-insecure)",
					"--result-file-image-digest",
					"$(results.shp-image-digest.path)",
					"--result-file-image-size",
					"$(results.shp-image-size.path)",
					"--secret-path",
					"/workspace/shp-push-secret",
				}))
				Expect(processedTaskRun.Spec.TaskSpec.Steps[1].VolumeMounts).To(utils.ContainNamedElement("shp-output-directory"))
				Expect(processedTaskRun.Spec.TaskSpec.Steps[1].VolumeMounts).To(utils.ContainNamedElement("shp-some-secret"))
			})
		})
	})
})

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/sources"

	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("HTTP", func() {

	cfg := config.NewDefaultConfig()

	Context("when a TaskSpec does not contain an step", func() {	
		var taskSpec *tektonv1beta1.TaskSpec

		BeforeEach(func() {
			taskSpec = &tektonv1beta1.TaskSpec{}
		})

		It("adds the first step", func() {
			sources.AppendHTTPStep(cfg, taskSpec, buildv1alpha1.BuildSource{
				Name: "logo",
				URL:  "https://shipwright.io/icons/logo.svg",
			})

			Expect(len(taskSpec.Steps)).To(Equal(1))
			Expect(taskSpec.Steps[0].Name).To(Equal(sources.RemoteArtifactsContainerName))
			Expect(taskSpec.Steps[0].Image).To(Equal(cfg.RemoteArtifactsContainerImage))
			Expect(taskSpec.Steps[0].WorkingDir).To(Equal("$(params.shp-source-root)"))
			Expect(taskSpec.Steps[0].Args[3]).To(Equal("wget \"https://shipwright.io/icons/logo.svg\""))
		})
	})

	Context("when a TaskSpec already contains the http step", func() {	
		var taskSpec *tektonv1beta1.TaskSpec

		BeforeEach(func() {
			taskSpec = &tektonv1beta1.TaskSpec{
				Steps: []tektonv1beta1.Step{
					{
						Container: corev1.Container{
							Name:       sources.RemoteArtifactsContainerName,
							Image:      cfg.RemoteArtifactsContainerImage,
							WorkingDir: "$(params.shp-source-root)",
							Command: []string{
								"/bin/sh",
							},
							Args: []string{
								"-e",
								"-x",
								"-c",
								"wget \"https://tekton.dev/images/tekton-horizontal-color.png\"",
							},
						},
					},
				},
			}
		})

		It("updates the existing step", func() {
			sources.AppendHTTPStep(cfg, taskSpec, buildv1alpha1.BuildSource{
				Name: "logo",
				URL:  "https://shipwright.io/icons/logo.svg",
			})

			Expect(len(taskSpec.Steps)).To(Equal(1))
			Expect(taskSpec.Steps[0].Name).To(Equal(sources.RemoteArtifactsContainerName))
			Expect(taskSpec.Steps[0].Image).To(Equal(cfg.RemoteArtifactsContainerImage))
			Expect(taskSpec.Steps[0].WorkingDir).To(Equal("$(params.shp-source-root)"))
			Expect(taskSpec.Steps[0].Args[3]).To(Equal("wget \"https://tekton.dev/images/tekton-horizontal-color.png\" ; wget \"https://shipwright.io/icons/logo.svg\""))
		})
	})

	Context("when a TaskSpec already another source step step", func() {	
		var taskSpec *tektonv1beta1.TaskSpec

		BeforeEach(func() {
			taskSpec = &tektonv1beta1.TaskSpec{
				Steps: []tektonv1beta1.Step{
					{
						Container: corev1.Container{
							Name: "source-something",
						},
					},
				},
			}
		})

		It("appends the http step", func() {
			sources.AppendHTTPStep(cfg, taskSpec, buildv1alpha1.BuildSource{
				Name: "logo",
				URL:  "https://shipwright.io/icons/logo.svg",
			})

			Expect(len(taskSpec.Steps)).To(Equal(2))
			Expect(taskSpec.Steps[1].Name).To(Equal(sources.RemoteArtifactsContainerName))
			Expect(taskSpec.Steps[1].Image).To(Equal(cfg.RemoteArtifactsContainerImage))
			Expect(taskSpec.Steps[1].WorkingDir).To(Equal("$(params.shp-source-root)"))
			Expect(taskSpec.Steps[1].Args[3]).To(Equal("wget \"https://shipwright.io/icons/logo.svg\""))
		})
	})
})

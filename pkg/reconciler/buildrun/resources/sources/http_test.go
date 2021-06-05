// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/sources"

	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("HTTP", func() {

	cfg := config.NewDefaultConfig()

	When("a TaskSpec does not contain an step", func() {

		var taskSpec *tektonv1beta1.TaskSpec

		BeforeEach(func() {
			taskSpec = &tektonv1beta1.TaskSpec{}
		})

		It("adds the first step", func() {
			err := sources.AppendHTTPStep(cfg, taskSpec, buildv1alpha1.BuildSource{
				Name: "logo",
				Type: buildv1alpha1.BuildSourceTypeHTTP,
				HTTP: &buildv1alpha1.HTTPBuildSource{
					URL: "https://shipwright.io/icons/logo.svg",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(taskSpec.Steps)).To(Equal(1))
			Expect(taskSpec.Steps[0].Name).To(Equal("source-logo"))
			Expect(taskSpec.Steps[0].Image).To(Equal(cfg.RemoteArtifactsContainerImage))
			Expect(taskSpec.Steps[0].WorkingDir).To(Equal("$(params.shp-source-root)"))
			Expect(taskSpec.Steps[0].Args[3]).To(Equal("wget \"https://shipwright.io/icons/logo.svg\""))
		})
	})

	When("a TaskSpec already another source step", func() {

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
			err := sources.AppendHTTPStep(cfg, taskSpec, buildv1alpha1.BuildSource{
				Name: "logo",
				Type: buildv1alpha1.BuildSourceTypeHTTP,
				HTTP: &buildv1alpha1.HTTPBuildSource{
					URL: "https://shipwright.io/icons/logo.svg",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(taskSpec.Steps)).To(Equal(2))
			Expect(taskSpec.Steps[1].Name).To(Equal("source-logo"))
			Expect(taskSpec.Steps[1].Image).To(Equal(cfg.RemoteArtifactsContainerImage))
			Expect(taskSpec.Steps[1].WorkingDir).To(Equal("$(params.shp-source-root)"))
			Expect(taskSpec.Steps[1].Args[3]).To(Equal("wget \"https://shipwright.io/icons/logo.svg\""))
		})
	})

	When("a destination directory is specified", func() {
		var taskSpec *tektonv1beta1.TaskSpec

		BeforeEach(func() {
			taskSpec = &tektonv1beta1.TaskSpec{}
		})

		It("uses the destination as the container working directory", func() {
			err := sources.AppendHTTPStep(cfg, taskSpec, buildv1alpha1.BuildSource{
				Name: "logo",
				Type: buildv1alpha1.BuildSourceTypeHTTP,
				HTTP: &buildv1alpha1.HTTPBuildSource{
					URL: "https://shipwright.io/icons/logo.svg",
				},
				Destination: "logo",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(taskSpec.Steps)).To(Equal(1))
			Expect(taskSpec.Steps[0].WorkingDir).To(Equal("$(params.shp-source-root)/logo"))
		})
	})
})

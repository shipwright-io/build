// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/sources"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"k8s.io/utils/ptr"
)

var _ = Describe("Git", func() {

	cfg := config.NewDefaultConfig()

	Context("when adding a public Git source", func() {
		var taskSpec *pipelineapi.TaskSpec

		BeforeEach(func() {
			taskSpec = &pipelineapi.TaskSpec{}
		})

		JustBeforeEach(func() {
			sources.AppendGitStep(cfg, taskSpec, buildv1beta1.Git{
				URL: "https://github.com/shipwright-io/build",
			}, "default")
		})

		It("adds results for the commit sha, commit author and branch name", func() {
			Expect(len(taskSpec.Results)).To(Equal(3))
			Expect(taskSpec.Results[0].Name).To(Equal("shp-source-default-commit-sha"))
			Expect(taskSpec.Results[1].Name).To(Equal("shp-source-default-commit-author"))
			Expect(taskSpec.Results[2].Name).To(Equal("shp-source-default-branch-name"))
		})

		It("adds a step", func() {
			Expect(len(taskSpec.Steps)).To(Equal(1))
			Expect(taskSpec.Steps[0].Name).To(Equal("source-default"))
			Expect(taskSpec.Steps[0].Image).To(Equal(cfg.GitContainerTemplate.Image))
			Expect(taskSpec.Steps[0].Args).To(Equal([]string{
				"--url", "https://github.com/shipwright-io/build",
				"--target", "$(params.shp-source-root)",
				"--result-file-commit-sha", "$(results.shp-source-default-commit-sha.path)",
				"--result-file-commit-author", "$(results.shp-source-default-commit-author.path)",
				"--result-file-branch-name", "$(results.shp-source-default-branch-name.path)",
				"--result-file-error-message", "$(results.shp-error-message.path)",
				"--result-file-error-reason", "$(results.shp-error-reason.path)",
				"--result-file-source-timestamp", "$(results.shp-source-default-source-timestamp.path)",
			}))
		})
	})

	Context("when adding a private Git source", func() {
		var taskSpec *pipelineapi.TaskSpec

		BeforeEach(func() {
			taskSpec = &pipelineapi.TaskSpec{}
		})

		JustBeforeEach(func() {
			sources.AppendGitStep(cfg, taskSpec, buildv1beta1.Git{
				URL:         "git@github.com:shipwright-io/build.git",
				CloneSecret: ptr.To("a.secret"),
			}, "default")
		})

		It("adds results for the commit sha, commit author and branch name", func() {
			Expect(len(taskSpec.Results)).To(Equal(3))
			Expect(taskSpec.Results[0].Name).To(Equal("shp-source-default-commit-sha"))
			Expect(taskSpec.Results[1].Name).To(Equal("shp-source-default-commit-author"))
			Expect(taskSpec.Results[2].Name).To(Equal("shp-source-default-branch-name"))
		})

		It("adds a volume for the secret", func() {
			Expect(len(taskSpec.Volumes)).To(Equal(3))
			Expect(taskSpec.Volumes[0].Name).To(Equal("shp-a-secret"))
			Expect(taskSpec.Volumes[0].VolumeSource.Secret).NotTo(BeNil())
			Expect(taskSpec.Volumes[0].VolumeSource.Secret.SecretName).To(Equal("a.secret"))
		})

		It("adds a step", func() {
			Expect(len(taskSpec.Steps)).To(Equal(1))
			Expect(taskSpec.Steps[0].Name).To(Equal("source-default"))
			Expect(taskSpec.Steps[0].Image).To(Equal(cfg.GitContainerTemplate.Image))
			Expect(taskSpec.Steps[0].Args).To(Equal([]string{
				"--url", "git@github.com:shipwright-io/build.git",
				"--target", "$(params.shp-source-root)",
				"--result-file-commit-sha", "$(results.shp-source-default-commit-sha.path)",
				"--result-file-commit-author", "$(results.shp-source-default-commit-author.path)",
				"--result-file-branch-name", "$(results.shp-source-default-branch-name.path)",
				"--result-file-error-message", "$(results.shp-error-message.path)",
				"--result-file-error-reason", "$(results.shp-error-reason.path)",
				"--result-file-source-timestamp", "$(results.shp-source-default-source-timestamp.path)",
				"--secret-path", "/workspace/shp-source-secret",
			}))
			Expect(len(taskSpec.Steps[0].VolumeMounts)).To(Equal(3))
			Expect(taskSpec.Steps[0].VolumeMounts[0].Name).To(Equal("shp-a-secret"))
			Expect(taskSpec.Steps[0].VolumeMounts[0].MountPath).To(Equal("/workspace/shp-source-secret"))
			Expect(taskSpec.Steps[0].VolumeMounts[0].ReadOnly).To(BeTrue())
		})
	})

	Context("when adding a Git source with a depth parameter", func() {
		var taskSpec *pipelineapi.TaskSpec

		BeforeEach(func() {
			taskSpec = &pipelineapi.TaskSpec{}
		})

		It("adds --depth argument with value 0 when depth is set to 0", func() {
			depth := 0
			sources.AppendGitStep(cfg, taskSpec, buildv1beta1.Git{
				URL:   "https://github.com/shipwright-io/build",
				Depth: ptr.To(depth),
			}, "default")

			Expect(len(taskSpec.Steps)).To(Equal(1))
			// Check specific arguments for --depth 0
			Expect(taskSpec.Steps[0].Args).To(ContainElement("--depth"))
			Expect(taskSpec.Steps[0].Args).To(ContainElement("0"))
			// Ensure other necessary args are still present
			Expect(taskSpec.Steps[0].Args).To(ContainElement("--url"))
		})

		It("adds --depth argument with value 1 when depth is set to 1", func() {
			depth := 1
			sources.AppendGitStep(cfg, taskSpec, buildv1beta1.Git{
				URL:   "https://github.com/shipwright-io/build",
				Depth: ptr.To(depth),
			}, "default")

			Expect(len(taskSpec.Steps)).To(Equal(1))
			Expect(taskSpec.Steps[0].Args).To(ContainElements("--depth", "1"))
		})

		It("adds --depth argument with specified value when depth is greater than 1", func() {
			depth := 5
			sources.AppendGitStep(cfg, taskSpec, buildv1beta1.Git{
				URL:   "https://github.com/shipwright-io/build",
				Depth: ptr.To(depth),
			}, "default")

			Expect(len(taskSpec.Steps)).To(Equal(1))
			Expect(taskSpec.Steps[0].Args).To(ContainElements("--depth", "5"))
		})

		It("does not add --depth argument when source.Depth is nil (not specified)", func() {
			sources.AppendGitStep(cfg, taskSpec, buildv1beta1.Git{
				URL: "https://github.com/shipwright-io/build",
			}, "default")

			Expect(len(taskSpec.Steps)).To(Equal(1))
			Expect(taskSpec.Steps[0].Args).NotTo(ContainElement("--depth"))
		})

		It("combines depth with other Git parameters", func() {
			depth := 3
			revision := "main"
			sources.AppendGitStep(cfg, taskSpec, buildv1beta1.Git{
				URL:         "https://github.com/shipwright-io/build",
				Depth:       ptr.To(depth),
				Revision:    ptr.To(revision),
				CloneSecret: ptr.To("a.secret"),
			}, "default")

			Expect(len(taskSpec.Steps)).To(Equal(1))
			Expect(taskSpec.Steps[0].Args).To(ContainElements(
				"--url", "https://github.com/shipwright-io/build",
				"--revision", "main",
				"--depth", "3",
				"--secret-path", "/workspace/shp-source-secret",
			))
		})

		It("correctly passes --depth 0 when combined with other Git parameters", func() {
			depth := 0
			revision := "develop"
			sources.AppendGitStep(cfg, taskSpec, buildv1beta1.Git{
				URL:         "https://github.com/shipwright-io/another-repo",
				Depth:       ptr.To(depth),
				Revision:    ptr.To(revision),
				CloneSecret: ptr.To("another.secret"),
			}, "default")

			Expect(len(taskSpec.Steps)).To(Equal(1))
			Expect(taskSpec.Steps[0].Args).To(ContainElements(
				"--url", "https://github.com/shipwright-io/another-repo",
				"--revision", "develop",
				"--depth", "0",
				"--secret-path", "/workspace/shp-source-secret",
			))
			Expect(taskSpec.Volumes).To(ContainElement(HaveField("Name", "shp-another-secret")))
			Expect(taskSpec.Steps[0].VolumeMounts).To(ContainElement(HaveField("Name", "shp-another-secret")))
		})
	})
})

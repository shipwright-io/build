// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/sources"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Utils", func() {
	Context("for different candidate volume names", func() {
		It("adds only the prefix if the name is okay", func() {
			Expect(sources.SanitizeVolumeNameForSecretName("okay-name")).To(Equal("shp-okay-name"))
		})

		It("adds the prefix and replaces characters that are not allowed", func() {
			Expect(sources.SanitizeVolumeNameForSecretName("bad.name")).To(Equal("shp-bad-name"))
		})

		It("adds the prefix and reduces the length if needed", func() {
			Expect(sources.SanitizeVolumeNameForSecretName("long-name-long-name-long-name-long-name-long-name-long-name-long-name-")).To(Equal("shp-long-name-long-name-long-name-long-name-long-name-long-name"))
		})

		It("ensures that the volume name ends with an alpha-numeric character", func() {
			// "shp-" + "abcdefghijklmnopqrstuvwxyz-abcdefghijklmnopqrstuvwxyz-abcd-efgh" reduced to 63 characters would be "shp-abcdefghijklmnopqrstuvwxyz-abcdefghijklmnopqrstuvwxyz-abcd-"
			Expect(sources.SanitizeVolumeNameForSecretName("abcdefghijklmnopqrstuvwxyz-abcdefghijklmnopqrstuvwxyz-abcd-efgh")).To(Equal("shp-abcdefghijklmnopqrstuvwxyz-abcdefghijklmnopqrstuvwxyz-abcd"))
		})
	})

	Context("when a TaskSpec does not contain any volume", func() {
		var taskSpec *pipelineapi.TaskSpec

		BeforeEach(func() {
			taskSpec = &pipelineapi.TaskSpec{}
		})

		It("adds the first volume", func() {
			sources.AppendSecretVolume(taskSpec, "a-secret")

			Expect(len(taskSpec.Volumes)).To(Equal(1))
			Expect(taskSpec.Volumes[0].Name).To(Equal("shp-a-secret"))
			Expect(taskSpec.Volumes[0].VolumeSource.Secret).NotTo(BeNil())
			Expect(taskSpec.Volumes[0].VolumeSource.Secret.SecretName).To(Equal("a-secret"))
		})
	})

	Context("when a TaskSpec already contains a volume secret", func() {
		var taskSpec *pipelineapi.TaskSpec

		BeforeEach(func() {
			taskSpec = &pipelineapi.TaskSpec{
				Volumes: []corev1.Volume{
					{
						Name: "shp-a-secret",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "a-secret",
							},
						},
					},
				},
			}
		})

		It("adds another one when the name does not match", func() {
			sources.AppendSecretVolume(taskSpec, "b-secret")

			Expect(len(taskSpec.Volumes)).To(Equal(2))
			Expect(taskSpec.Volumes[1].Name).To(Equal("shp-b-secret"))
			Expect(taskSpec.Volumes[1].VolumeSource.Secret).NotTo(BeNil())
			Expect(taskSpec.Volumes[1].VolumeSource.Secret.SecretName).To(Equal("b-secret"))
		})

		It("keeps the volume list unchanged if the same secret is appended", func() {
			sources.AppendSecretVolume(taskSpec, "a-secret")

			Expect(len(taskSpec.Volumes)).To(Equal(1))
		})
	})

	Context("SetupHomeAndTmpVolumes", func() {
		var taskSpec *pipelineapi.TaskSpec
		var targetStep *pipelineapi.Step

		BeforeEach(func() {
			taskSpec = &pipelineapi.TaskSpec{}
			targetStep = &pipelineapi.Step{
				Name: "test-step",
				Env:  []corev1.EnvVar{},
			}
		})

		It("creates volumes and mounts for HOME and TMPDIR", func() {
			sources.SetupHomeAndTmpVolumes(taskSpec, targetStep)

			// Verify that two volumes were created
			Expect(len(taskSpec.Volumes)).To(Equal(2))

			// Verify volume types are EmptyDir
			Expect(taskSpec.Volumes[0].VolumeSource.EmptyDir).NotTo(BeNil())
			Expect(taskSpec.Volumes[1].VolumeSource.EmptyDir).NotTo(BeNil())

			// Verify volume names start with expected prefixes
			Expect(taskSpec.Volumes[0].Name).To(ContainSubstring("shp-tmp-"))
			Expect(taskSpec.Volumes[1].Name).To(ContainSubstring("shp-home-"))

			// Verify volume mounts were added to the step
			Expect(len(targetStep.VolumeMounts)).To(Equal(2))

			// Verify mount paths
			Expect(targetStep.VolumeMounts[0].MountPath).To(Equal("/shp-tmp"))
			Expect(targetStep.VolumeMounts[1].MountPath).To(Equal("/shp-writable-home"))

			// Verify environment variables were set
			Expect(len(targetStep.Env)).To(Equal(2))
			Expect(targetStep.Env[0].Name).To(Equal("TMPDIR"))
			Expect(targetStep.Env[0].Value).To(Equal("/shp-tmp"))
			Expect(targetStep.Env[1].Name).To(Equal("HOME"))
			Expect(targetStep.Env[1].Value).To(Equal("/shp-writable-home"))
		})

		It("overrides existing environment variables", func() {
			// Set up existing environment variables
			targetStep.Env = []corev1.EnvVar{
				{Name: "HOME", Value: "/original/home"},
				{Name: "TMPDIR", Value: "/original/tmp"},
				{Name: "OTHER_VAR", Value: "other-value"},
			}

			sources.SetupHomeAndTmpVolumes(taskSpec, targetStep)

			// Verify that existing env vars were overridden
			Expect(len(targetStep.Env)).To(Equal(3))
			// The original order is preserved, but values are overridden
			Expect(targetStep.Env[0].Name).To(Equal("HOME"))
			Expect(targetStep.Env[0].Value).To(Equal("/shp-writable-home"))
			Expect(targetStep.Env[1].Name).To(Equal("TMPDIR"))
			Expect(targetStep.Env[1].Value).To(Equal("/shp-tmp"))
			Expect(targetStep.Env[2].Name).To(Equal("OTHER_VAR"))
			Expect(targetStep.Env[2].Value).To(Equal("other-value"))
		})

		It("does not duplicate volumes when called multiple times", func() {
			// Call the function twice
			sources.SetupHomeAndTmpVolumes(taskSpec, targetStep)
			sources.SetupHomeAndTmpVolumes(taskSpec, targetStep)

			// Should still only have 2 volumes
			Expect(len(taskSpec.Volumes)).To(Equal(2))
			Expect(len(targetStep.VolumeMounts)).To(Equal(2))
		})

		It("handles step names with special characters", func() {
			targetStep.Name = "step-with.special@chars!"

			sources.SetupHomeAndTmpVolumes(taskSpec, targetStep)

			// Verify volumes were created despite special characters
			Expect(len(taskSpec.Volumes)).To(Equal(2))
			Expect(len(targetStep.VolumeMounts)).To(Equal(2))

			// Verify environment variables are still set correctly
			Expect(targetStep.Env[0].Name).To(Equal("TMPDIR"))
			Expect(targetStep.Env[0].Value).To(Equal("/shp-tmp"))
			Expect(targetStep.Env[1].Name).To(Equal("HOME"))
			Expect(targetStep.Env[1].Value).To(Equal("/shp-writable-home"))
		})

		It("works with existing volumes in TaskSpec", func() {
			// Add an existing volume
			taskSpec.Volumes = []corev1.Volume{
				{
					Name: "existing-volume",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			}

			sources.SetupHomeAndTmpVolumes(taskSpec, targetStep)

			// Should have 3 volumes total (1 existing + 2 new)
			Expect(len(taskSpec.Volumes)).To(Equal(3))
			Expect(len(targetStep.VolumeMounts)).To(Equal(2))
		})

		It("works with existing volume mounts in step", func() {
			// Add an existing volume mount
			targetStep.VolumeMounts = []corev1.VolumeMount{
				{
					Name:      "existing-mount",
					MountPath: "/existing/path",
				},
			}

			sources.SetupHomeAndTmpVolumes(taskSpec, targetStep)

			// Should have 3 volume mounts total (1 existing + 2 new)
			Expect(len(targetStep.VolumeMounts)).To(Equal(3))
			Expect(len(taskSpec.Volumes)).To(Equal(2))
		})
	})
})

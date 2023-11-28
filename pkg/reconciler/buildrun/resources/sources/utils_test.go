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
})

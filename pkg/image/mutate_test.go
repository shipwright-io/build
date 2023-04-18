// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/shipwright-io/build/pkg/image"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MutateImageOrImageIndex", func() {

	Context("for an image that has no annotations or labels", func() {

		var img containerreg.Image

		BeforeEach(func() {
			var err error
			img, err = random.Image(1234, 1)
			Expect(err).ToNot(HaveOccurred())
		})

		It("correctly adds labels and annotations", func() {
			newImg, newImageIndex, err := image.MutateImageOrImageIndex(img, nil,
				map[string]string{
					"annotation1": "someValue",
				},
				map[string]string{
					"label": "someLabelValue",
				})

			Expect(err).ToNot(HaveOccurred())
			Expect(newImageIndex).To(BeNil())
			Expect(newImg).ToNot(BeNil())
			Expect(img).ToNot(Equal(newImg))

			manifest, err := newImg.Manifest()
			Expect(err).ToNot(HaveOccurred())
			Expect(manifest.Annotations).To(HaveLen(1))
			Expect(manifest.Annotations["annotation1"]).To(Equal("someValue"))

			configFile, err := newImg.ConfigFile()
			Expect(err).ToNot(HaveOccurred())
			Expect(configFile.Config.Labels).To(HaveLen(1))
			Expect(configFile.Config.Labels["label"]).To(Equal("someLabelValue"))
		})
	})

	Context("For an image that has annotations and labels", func() {

		var img containerreg.Image

		BeforeEach(func() {
			var err error
			img, err = random.Image(1234, 1)
			Expect(err).ToNot(HaveOccurred())
			img = mutate.Annotations(img,
				map[string]string{
					"existingAnnotation1": "initialValue1",
					"existingAnnotation2": "initialValue2",
				}).(containerreg.Image)

			cfg, err := img.ConfigFile()
			Expect(err).ToNot(HaveOccurred())
			cfg = cfg.DeepCopy()
			cfg.Config.Labels = map[string]string{
				"existingLabel1": "initialValue1",
				"existingLabel2": "initialValue2",
			}
			img, err = mutate.ConfigFile(img, cfg)
			Expect(err).ToNot(HaveOccurred())
		})

		It("correctly adds and overwrites labels and annotations", func() {
			newImg, newImageIndex, err := image.MutateImageOrImageIndex(img, nil,
				map[string]string{
					"annotation1":         "someValue",
					"existingAnnotation1": "newValue",
				},
				map[string]string{
					"label":          "someLabelValue",
					"existingLabel1": "newValue",
				})

			Expect(err).ToNot(HaveOccurred())
			Expect(newImageIndex).To(BeNil())
			Expect(newImg).ToNot(BeNil())
			Expect(img).ToNot(Equal(newImg))

			manifest, err := newImg.Manifest()
			Expect(err).ToNot(HaveOccurred())
			Expect(manifest.Annotations).To(HaveLen(3))
			Expect(manifest.Annotations["annotation1"]).To(Equal("someValue"))
			Expect(manifest.Annotations["existingAnnotation1"]).To(Equal("newValue"))
			Expect(manifest.Annotations["existingAnnotation2"]).To(Equal("initialValue2"))

			configFile, err := newImg.ConfigFile()
			Expect(err).ToNot(HaveOccurred())
			Expect(configFile.Config.Labels).To(HaveLen(3))
			Expect(configFile.Config.Labels["label"]).To(Equal("someLabelValue"))
			Expect(configFile.Config.Labels["existingLabel1"]).To(Equal("newValue"))
			Expect(configFile.Config.Labels["existingLabel2"]).To(Equal("initialValue2"))
		})
	})

	Context("for an index that has no annotations or labels", func() {

		var index containerreg.ImageIndex

		BeforeEach(func() {
			var err error
			index, err = random.Index(4091, 2, 2)
			Expect(err).ToNot(HaveOccurred())
		})

		It("correctly adds labels and annotations to the index and the images", func() {
			newImg, newImageIndex, err := image.MutateImageOrImageIndex(nil, index,
				map[string]string{
					"annotation1": "someValue",
				},
				map[string]string{
					"label": "someLabelValue",
				})

			Expect(err).ToNot(HaveOccurred())
			Expect(newImageIndex).ToNot(BeNil())
			Expect(newImg).To(BeNil())
			Expect(index).ToNot(Equal(newImageIndex))

			// verify the annotations of the index
			indexManifest, err := newImageIndex.IndexManifest()
			Expect(err).ToNot(HaveOccurred())
			Expect(indexManifest.Annotations).To(HaveLen(1))
			Expect(indexManifest.Annotations["annotation1"]).To(Equal("someValue"))

			// verify the images
			Expect(indexManifest.Manifests).To(HaveLen(2))
			for _, descriptor := range indexManifest.Manifests {
				img, err := newImageIndex.Image(descriptor.Digest)
				Expect(err).ToNot(HaveOccurred())

				manifest, err := img.Manifest()
				Expect(err).ToNot(HaveOccurred())
				Expect(manifest.Annotations).To(HaveLen(1))
				Expect(manifest.Annotations["annotation1"]).To(Equal("someValue"))

				configFile, err := img.ConfigFile()
				Expect(err).ToNot(HaveOccurred())
				Expect(configFile.Config.Labels).To(HaveLen(1))
				Expect(configFile.Config.Labels["label"]).To(Equal("someLabelValue"))
			}
		})
	})
})

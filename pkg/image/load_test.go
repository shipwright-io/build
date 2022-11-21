// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	"os"
	"path"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/shipwright-io/build/pkg/image"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("For a remote repository", func() {

	Context("that does not exist", func() {

		imageName, err := name.ParseReference("ghcr.io/shipwright-io/non-existing:latest")
		Expect(err).ToNot(HaveOccurred())

		It("LoadImageOrImageIndexFromRegistry returns an error", func() {
			img, imageIndex, err := image.LoadImageOrImageIndexFromRegistry(imageName, []remote.Option{})
			Expect(err).To(HaveOccurred())
			Expect(imageIndex).To(BeNil())
			Expect(img).To(BeNil())
		})
	})

	Context("that contains a multi-platform image", func() {

		imageName, err := name.ParseReference("ghcr.io/shipwright-io/base-git:latest")
		Expect(err).ToNot(HaveOccurred())

		It("LoadImageOrImageIndexFromRegistry returns an ImageIndex", func() {
			img, imageIndex, err := image.LoadImageOrImageIndexFromRegistry(imageName, []remote.Option{})
			Expect(err).ToNot(HaveOccurred())
			Expect(imageIndex).ToNot(BeNil())
			Expect(img).To(BeNil())
		})
	})

	Context("that contains a single image", func() {

		imageName, err := name.ParseReference("ghcr.io/shipwright-io/sample-go/source-bundle:latest")
		Expect(err).ToNot(HaveOccurred())

		It("LoadImageOrImageIndexFromRegistry returns an Image", func() {
			img, imageIndex, err := image.LoadImageOrImageIndexFromRegistry(imageName, []remote.Option{})
			Expect(err).ToNot(HaveOccurred())
			Expect(imageIndex).To(BeNil())
			Expect(img).ToNot(BeNil())
		})
	})
})

var _ = Describe("For a local directory", func() {

	Context("that is empty", func() {

		var directory string

		BeforeEach(func() {
			directory, err := os.MkdirTemp(os.TempDir(), "empty")
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func() {
				err := os.RemoveAll(directory)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		It("LoadImageOrImageIndexFromDirectory returns an error", func() {
			img, imageIndex, err := image.LoadImageOrImageIndexFromDirectory(directory)
			Expect(err).To(HaveOccurred())
			Expect(imageIndex).To(BeNil())
			Expect(img).To(BeNil())
		})
	})

	Context("that contains a multi-platform image", func() {

		cwd, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		directory := path.Clean(path.Join(cwd, "../..", "test/data/images/multi-platform-image-in-oci"))

		It("LoadImageOrImageIndexFromDirectory returns an ImageIndex", func() {
			img, imageIndex, err := image.LoadImageOrImageIndexFromDirectory(directory)
			Expect(err).ToNot(HaveOccurred())
			Expect(imageIndex).ToNot(BeNil())
			Expect(img).To(BeNil())
		})
	})

	Context("that contains a single tar file with an image", func() {

		cwd, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		directory := path.Clean(path.Join(cwd, "../..", "test/data/images/single-image"))

		It("LoadImageOrImageIndexFromDirectory returns an Image", func() {
			img, imageIndex, err := image.LoadImageOrImageIndexFromDirectory(directory)
			Expect(err).ToNot(HaveOccurred())
			Expect(imageIndex).To(BeNil())
			Expect(img).ToNot(BeNil())
		})
	})

	Context("that contains an OCI image layout with a single image", func() {

		cwd, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		directory := path.Clean(path.Join(cwd, "../..", "test/data/images/single-image-in-oci"))

		It("LoadImageOrImageIndexFromDirectory returns an Image", func() {
			img, imageIndex, err := image.LoadImageOrImageIndexFromDirectory(directory)
			Expect(err).ToNot(HaveOccurred())
			Expect(imageIndex).To(BeNil())
			Expect(img).ToNot(BeNil())
		})
	})
})

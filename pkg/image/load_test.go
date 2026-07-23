// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"net/url"
	"os"
	"path"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/shipwright-io/build/pkg/image"
)

var _ = Describe("For a remote repository", func() {
	withTempRegistry := func(f func(endpoint string)) {
		s := httptest.NewServer(
			registry.New(
				registry.Logger(log.New(io.Discard, "", 0)),
			),
		)
		defer s.Close()

		u, err := url.Parse(s.URL)
		Expect(err).ToNot(HaveOccurred())

		f(u.Host)
	}

	Context("that does not exist", func() {
		It("LoadImageOrImageIndexFromRegistry returns an error", func() {
			withTempRegistry(func(endpoint string) {
				imageName, err := name.ParseReference(fmt.Sprintf("%s/namespace/non-existing-%s:latest", endpoint, rand.String(5)))
				Expect(err).ToNot(HaveOccurred())

				img, imageIndex, err := image.LoadImageOrImageIndexFromRegistry(imageName, []remote.Option{})
				Expect(err).To(HaveOccurred())
				Expect(imageIndex).To(BeNil())
				Expect(img).To(BeNil())
			})
		})
	})

	Context("that contains a multi-platform image", func() {
		It("LoadImageOrImageIndexFromRegistry returns an ImageIndex", func() {
			withTempRegistry(func(endpoint string) {
				imageName, err := name.ParseReference(fmt.Sprintf("%s/namespace/multi-platform-%s:latest", endpoint, rand.String(5)))
				Expect(err).ToNot(HaveOccurred())

				index, err := random.Index(1234, 1, 2)
				Expect(err).ToNot(HaveOccurred())
				Expect(remote.WriteIndex(imageName, index)).To(Succeed())

				img, imageIndex, err := image.LoadImageOrImageIndexFromRegistry(imageName, []remote.Option{})
				Expect(err).ToNot(HaveOccurred())
				Expect(imageIndex).ToNot(BeNil())
				Expect(img).To(BeNil())
			})
		})
	})

	Context("that contains a single image", func() {
		It("LoadImageOrImageIndexFromRegistry returns an Image", func() {
			withTempRegistry(func(endpoint string) {
				imageName, err := name.ParseReference(fmt.Sprintf("%s/namespace/single-image-%s:latest", endpoint, rand.String(5)))
				Expect(err).ToNot(HaveOccurred())

				Expect(remote.Write(imageName, empty.Image)).To(Succeed())

				img, imageIndex, err := image.LoadImageOrImageIndexFromRegistry(imageName, []remote.Option{})
				Expect(err).ToNot(HaveOccurred())
				Expect(imageIndex).To(BeNil())
				Expect(img).ToNot(BeNil())
			})
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
			img, imageIndex, isImageFromTar, err := image.LoadImageOrImageIndexFromDirectory(directory)
			Expect(err).To(HaveOccurred())
			Expect(imageIndex).To(BeNil())
			Expect(img).To(BeNil())
			Expect(isImageFromTar).To(BeFalse())
		})
	})

	Context("that contains a multi-platform image", func() {

		cwd, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		directory := path.Clean(path.Join(cwd, "../..", "test/data/images/multi-platform-image-in-oci"))

		It("LoadImageOrImageIndexFromDirectory returns an ImageIndex", func() {
			img, imageIndex, isImageFromTar, err := image.LoadImageOrImageIndexFromDirectory(directory)
			Expect(err).ToNot(HaveOccurred())
			Expect(imageIndex).ToNot(BeNil())
			Expect(img).To(BeNil())
			Expect(isImageFromTar).To(BeFalse())
		})
	})

	Context("that contains a single tar file with an image", func() {

		cwd, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		directory := path.Clean(path.Join(cwd, "../..", "test/data/images/single-image"))

		It("LoadImageOrImageIndexFromDirectory returns an Image", func() {
			img, imageIndex, isImageFromTar, err := image.LoadImageOrImageIndexFromDirectory(directory)
			Expect(err).ToNot(HaveOccurred())
			Expect(imageIndex).To(BeNil())
			Expect(img).ToNot(BeNil())
			Expect(isImageFromTar).To(BeTrue())
		})
	})

	Context("that contains an OCI image layout with a single image", func() {

		cwd, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		directory := path.Clean(path.Join(cwd, "../..", "test/data/images/single-image-in-oci"))

		It("LoadImageOrImageIndexFromDirectory returns an Image", func() {
			img, imageIndex, isImageFromTar, err := image.LoadImageOrImageIndexFromDirectory(directory)
			Expect(err).ToNot(HaveOccurred())
			Expect(imageIndex).To(BeNil())
			Expect(img).ToNot(BeNil())
			Expect(isImageFromTar).To(BeFalse())
		})
	})
})

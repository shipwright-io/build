// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main_test

import (
	"context"
	"fmt"
	"log"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/shipwright-io/build/cmd/image-processing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/rand"
)

var _ = Describe("Image Processing Resource", func() {
	run := func(args ...string) error {
		log.SetOutput(GinkgoWriter)

		// `pflag.Parse()` parses the command-line flags from os.Args[1:]
		// appending `tool`(can be anything) at beginning of args array
		// to avoid trimming the args we pass
		os.Args = append([]string{"tool"}, args...)

		// Simulate 2>/dev/null redirect as of now, there is no test case
		// that checks the Stderr output of the command-line tool
		tmp := os.Stderr
		defer func() { os.Stderr = tmp }()
		os.Stderr = nil

		return Execute(context.Background())
	}

	AfterEach(func() {
		// Reset flag variables
		pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	})

	withTempRegistry := func(f func(endpoint string)) {
		logLogger := log.Logger{}
		logLogger.SetOutput(GinkgoWriter)

		s := httptest.NewServer(
			registry.New(
				registry.Logger(&logLogger),
				registry.WithReferrersSupport(true),
			),
		)
		defer s.Close()

		u, err := url.Parse(s.URL)
		Expect(err).ToNot(HaveOccurred())

		f(u.Host)
	}

	withTestImage := func(f func(tag name.Tag)) {
		withTempRegistry(func(endpoint string) {
			tag, err := name.NewTag(fmt.Sprintf("%s/%s:%s", endpoint, "temp-image", rand.String(5)))
			Expect(err).ToNot(HaveOccurred())

			Expect(remote.Write(tag, empty.Image)).To(Succeed())
			f(tag)
		})
	}

	getCompressedImageSize := func(img containerreg.Image) int64 {
		manifest, err := img.Manifest()
		Expect(err).ToNot(HaveOccurred())

		configSize := manifest.Config.Size

		var layersSize int64
		for _, layer := range manifest.Layers {
			layersSize += layer.Size
		}

		return layersSize + configSize
	}

	getImageConfigLabel := func(image, label string) string {
		ref, err := name.ParseReference(image)
		Expect(err).ToNot(HaveOccurred())

		img, err := remote.Image(ref)
		Expect(err).ToNot(HaveOccurred())

		config, err := img.ConfigFile()
		Expect(err).ToNot(HaveOccurred())

		return config.Config.Labels[label]
	}

	getImageAnnotation := func(image, annotation string) string {
		ref, err := name.ParseReference(image)
		Expect(err).ToNot(HaveOccurred())

		img, err := remote.Image(ref)
		Expect(err).ToNot(HaveOccurred())

		manifest, err := img.Manifest()
		Expect(err).ToNot(HaveOccurred())

		return manifest.Annotations[annotation]
	}

	withTempFile := func(pattern string, f func(filename string)) {
		file, err := os.CreateTemp(os.TempDir(), pattern)
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove(file.Name())

		f(file.Name())
	}

	filecontent := func(path string) string {
		data, err := os.ReadFile(path)
		Expect(err).ToNot(HaveOccurred())
		return string(data)
	}

	getImage := func(tag name.Tag) containerreg.Image {
		ref, err := name.ParseReference(tag.String())
		Expect(err).ToNot(HaveOccurred())

		desc, err := remote.Get(ref)
		Expect(err).ToNot(HaveOccurred())

		img, err := desc.Image()
		Expect(err).ToNot(HaveOccurred())

		return img
	}

	getImageDigest := func(tag name.Tag) containerreg.Hash {
		digest, err := getImage(tag).Digest()
		Expect(err).ToNot(HaveOccurred())

		return digest
	}

	Context("validations and error cases", func() {
		It("should succeed in case the help is requested", func() {
			Expect(run("--help")).ToNot(HaveOccurred())
		})

		It("should fail in case mandatory arguments are missing", func() {
			Expect(run()).To(HaveOccurred())
		})

		It("should fail in case --image is empty", func() {
			Expect(run("--image", "")).To(HaveOccurred())
		})

		It("should fail in case --image does not exist", func() {
			Expect(run(
				"--image", "docker.io/feqlQoDIHc/bcfHFHHXYF",
			)).To(HaveOccurred())
		})

		It("should fail in case annotation is invalid", func() {
			withTestImage(func(tag name.Tag) {
				Expect(run(
					"--image", tag.String(),
					"--annotation", "org.opencontainers.image.url*https://my-company.com/images",
				)).To(HaveOccurred())
			})
		})

		It("should fail in case label is invalid", func() {
			withTestImage(func(tag name.Tag) {
				Expect(run(
					"--image", tag.String(),
					"--label", " description*image description",
				)).To(HaveOccurred())
			})
		})
	})

	Context("mutating the image", func() {
		It("should mutate an image with single annotation", func() {
			withTestImage(func(tag name.Tag) {
				Expect(run(
					"--insecure",
					"--image", tag.String(),
					"--annotation", "org.opencontainers.image.url=https://my-company.com/images",
				)).ToNot(HaveOccurred())

				Expect(getImageAnnotation(tag.String(), "org.opencontainers.image.url")).
					To(Equal("https://my-company.com/images"))
			})
		})

		It("should mutate an image with multiple annotations", func() {
			withTestImage(func(tag name.Tag) {
				Expect(run(
					"--insecure",
					"--image", tag.String(),
					"--annotation", "org.opencontainers.image.url=https://my-company.com/images",
					"--annotation", "org.opencontainers.image.source=https://github.com/org/repo",
				)).ToNot(HaveOccurred())

				Expect(getImageAnnotation(tag.String(), "org.opencontainers.image.url")).
					To(Equal("https://my-company.com/images"))

				Expect(getImageAnnotation(tag.String(), "org.opencontainers.image.source")).
					To(Equal("https://github.com/org/repo"))
			})
		})

		It("should mutate an image with single label", func() {
			withTestImage(func(tag name.Tag) {
				Expect(run(
					"--insecure",
					"--image", tag.String(),
					"--label", "description=image description",
				)).ToNot(HaveOccurred())

				Expect(getImageConfigLabel(tag.String(), "description")).
					To(Equal("image description"))
			})
		})

		It("should mutate an image with multiple labels", func() {
			withTestImage(func(tag name.Tag) {
				Expect(run(
					"--insecure",
					"--image", tag.String(),
					"--label", "description=image description",
					"--label", "maintainer=team@my-company.com",
				)).ToNot(HaveOccurred())

				Expect(getImageConfigLabel(tag.String(), "description")).
					To(Equal("image description"))

				Expect(getImageConfigLabel(tag.String(), "maintainer")).
					To(Equal("team@my-company.com"))
			})
		})

		It("should mutate an image with both annotation and label", func() {
			withTestImage(func(tag name.Tag) {
				Expect(run(
					"--insecure",
					"--image", tag.String(),
					"--label", "description=image description",
					"--annotation", "org.opencontainers.image.url=https://my-company.com/images",
				)).ToNot(HaveOccurred())

				Expect(getImageConfigLabel(tag.String(), "description")).
					To(Equal("image description"))

				Expect(getImageAnnotation(tag.String(), "org.opencontainers.image.url")).
					To(Equal("https://my-company.com/images"))
			})
		})
	})

	Context("store result after image mutation", func() {
		It("should store image digest into file specified in --result-file-image-digest flags", func() {
			withTestImage(func(tag name.Tag) {
				withTempFile("image-digest", func(filename string) {
					Expect(run(
						"--insecure",
						"--image", tag.String(),
						"--annotation", "org.opencontainers.image.url=https://my-company.com/images",
						"--result-file-image-digest", filename,
					)).ToNot(HaveOccurred())

					Expect(filecontent(filename)).To(Equal(getImageDigest(tag).String()))
				})
			})
		})

		It("should store image size into file specified in result-file-image-size flags", func() {
			withTestImage(func(tag name.Tag) {
				withTempFile("image-size", func(filename string) {
					Expect(run(
						"--insecure",
						"--image", tag.String(),
						"--annotation", "org.opencontainers.image.url=https://my-company.com/images",
						"--result-file-image-size", filename,
					)).ToNot(HaveOccurred())

					size := getCompressedImageSize(getImage(tag))
					Expect(filecontent(filename)).To(Equal(strconv.FormatInt(size, 10)))
				})
			})
		})
	})
})

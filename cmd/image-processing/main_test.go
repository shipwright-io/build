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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/shipwright-io/build/cmd/image-processing"

	"github.com/google/go-containerregistry/pkg/crane"
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

	withTempDir := func(f func(target string)) {
		path, err := os.MkdirTemp(os.TempDir(), "temp-dir")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(path)

		f(path)
	}

	withTestImage := func(f func(tag name.Tag)) {
		withTempRegistry(func(endpoint string) {
			tag, err := name.NewTag(fmt.Sprintf("%s/%s:%s", endpoint, "temp-image", rand.String(5)))
			Expect(err).ToNot(HaveOccurred())

			Expect(remote.Write(tag, empty.Image)).To(Succeed())
			f(tag)
		})
	}

	withTestImageAsDirectory := func(f func(path string, tag name.Tag)) {
		withTempRegistry(func(endpoint string) {
			withTempDir(func(dir string) {
				tag, err := name.NewTag(fmt.Sprintf("%s/%s:%s", endpoint, "temp-image", rand.String(5)))
				Expect(err).ToNot(HaveOccurred())

				Expect(crane.SaveOCI(empty.Image, dir)).To(Succeed())

				f(dir, tag)
			})
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
			Expect(run()).ToNot(Succeed())
		})

		It("should fail in case --image is empty", func() {
			Expect(run(
				"--image", "",
			)).To(FailWith("argument must not be empty"))
		})

		It("should fail in case --image does not exist", func() {
			Expect(run(
				"--image", "docker.io/library/feqlqodihc:bcfhfhhxyf",
			)).To(FailWith("unexpected status code 401"))
		})

		It("should fail in case annotation is invalid", func() {
			withTestImage(func(tag name.Tag) {
				Expect(run(
					"--insecure",
					"--image", tag.String(),
					"--annotation", "org.opencontainers.image.url*https://my-company.com/images",
				)).To(FailWith("not enough parts"))
			})
		})

		It("should fail in case label is invalid", func() {
			withTestImage(func(tag name.Tag) {
				Expect(run(
					"--insecure",
					"--image", tag.String(),
					"--label", " description*image description",
				)).To(FailWith("not enough parts"))
			})
		})

		It("should fail if both --image-timestamp and --image-timestamp-file are used", func() {
			Expect(run(
				"--image-timestamp", "1234567890",
				"--image-timestamp-file", "/tmp/foobar",
			)).To(FailWith("image timestamp and image timestamp file flag is used"))
		})

		It("should fail if --image-timestamp-file is used with a non-existing file", func() {
			Expect("/tmp/does-not-exist").ToNot(BeAnExistingFile())
			Expect(run(
				"--image-timestamp-file", "/tmp/does-not-exist",
			)).To(FailWith("image timestamp file flag references a non-existing file"))
		})

		It("should fail if --image-timestamp-file referenced file cannot be used", func() {
			withTempDir(func(wrong string) {
				Expect(run(
					"--image-timestamp-file", wrong,
				)).To(FailWith("failed to read image timestamp from"))
			})
		})

		It("should fail in case timestamp is invalid", func() {
			withTestImage(func(tag name.Tag) {
				Expect(run(
					"--insecure",
					"--image", tag.String(),
					"--image-timestamp", "foobar",
				)).To(FailWith("failed to parse image timestamp"))
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

		It("should mutate the image timestamp using a provided timestamp", func() {
			withTestImageAsDirectory(func(path string, tag name.Tag) {
				Expect(run(
					"--insecure",
					"--push", path,
					"--image", tag.String(),
					"--image-timestamp", "1234567890",
				)).ToNot(HaveOccurred())

				image := getImage(tag)

				cfgFile, err := image.ConfigFile()
				Expect(err).ToNot(HaveOccurred())

				Expect(cfgFile.Created.Time).To(BeTemporally("==", time.Unix(1234567890, 0)))
			})
		})

		It("should mutate the image timestamp using a provided timestamp in a file", func() {
			withTestImageAsDirectory(func(path string, tag name.Tag) {
				withTempFile("timestamp", func(filename string) {
					Expect(os.WriteFile(filename, []byte("1234567890"), os.FileMode(0644)))

					Expect(run(
						"--insecure",
						"--push", path,
						"--image", tag.String(),
						"--image-timestamp-file", filename,
					)).ToNot(HaveOccurred())

					image := getImage(tag)

					cfgFile, err := image.ConfigFile()
					Expect(err).ToNot(HaveOccurred())

					Expect(cfgFile.Created.Time).To(BeTemporally("==", time.Unix(1234567890, 0)))
				})
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

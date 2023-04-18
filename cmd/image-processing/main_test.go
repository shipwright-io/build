// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"

	. "github.com/shipwright-io/build/cmd/image-processing"
)

var _ = Describe("Image Processing Resource", func() {
	run := func(args ...string) error {
		log.SetOutput(io.Discard)

		// `pflag.Parse()` parses the command-line flags from os.Args[1:]
		// appending `tool`(can be anything) at beginning of args array
		// to avoid trimming the args we pass
		os.Args = append([]string{"tool"}, args...)

		return Execute(context.Background())
	}

	const (
		regUser   = "REGISTRY_USERNAME"
		regPass   = "REGISTRY_PASSWORD"
		imageHost = "IMAGE_HOST"
		image     = "IMAGE"
	)

	BeforeEach(func() {
		for _, env := range []string{regUser, regPass, imageHost, image} {
			if _, ok := os.LookupEnv(env); !ok {
				Skip(fmt.Sprintf("Skipping test case, because environment variable %s is not set", env))
			}
		}
	})

	resetFlags := func() {
		// Reset flag variables
		pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	}

	AfterEach(func() {
		resetFlags()
	})

	imageURL := fmt.Sprintf("%s/%s", os.Getenv(imageHost), os.Getenv(image))

	pushImage := func(version string) name.Tag {
		auth := authn.FromConfig(authn.AuthConfig{
			Username: os.Getenv(regUser),
			Password: os.Getenv(regPass),
		})

		tag, err := name.NewTag(fmt.Sprintf("%s:%s", imageURL, version))
		Expect(err).ToNot(HaveOccurred())

		Expect(remote.Write(
			tag,
			empty.Image,
			remote.WithAuth(auth),
		)).ToNot(HaveOccurred())

		return tag
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
		auth := authn.FromConfig(authn.AuthConfig{
			Username: os.Getenv(regUser),
			Password: os.Getenv(regPass),
		})

		ref, err := name.ParseReference(image)
		Expect(err).ToNot(HaveOccurred())

		img, err := remote.Image(ref, remote.WithAuth(auth))
		Expect(err).ToNot(HaveOccurred())

		config, err := img.ConfigFile()
		Expect(err).ToNot(HaveOccurred())

		return config.Config.Labels[label]
	}

	getImageAnnotation := func(image, annotation string) string {
		auth := authn.FromConfig(authn.AuthConfig{
			Username: os.Getenv(regUser),
			Password: os.Getenv(regPass),
		})

		ref, err := name.ParseReference(image)
		Expect(err).ToNot(HaveOccurred())

		img, err := remote.Image(ref, remote.WithAuth(auth))
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

	withDockerConfigJSON := func(f func(dockerConfigJSONPath string)) {
		withTempFile("docker.config", func(tempFile string) {
			err := os.WriteFile(tempFile, ([]byte(fmt.Sprintf("{\"auths\":{%q:{\"username\":%q,\"password\":%q}}}", os.Getenv(imageHost), os.Getenv(regUser), os.Getenv(regPass)))), 0644)
			Expect(err).ToNot(HaveOccurred())

			f(tempFile)
		})
	}

	filecontent := func(path string) string {
		data, err := os.ReadFile(path)
		Expect(err).ToNot(HaveOccurred())
		return string(data)
	}

	getImage := func(tag name.Tag) containerreg.Image {
		auth := authn.FromConfig(authn.AuthConfig{
			Username: os.Getenv(regUser),
			Password: os.Getenv(regPass),
		})

		ref, err := name.ParseReference(tag.String())
		Expect(err).ToNot(HaveOccurred())

		desc, err := remote.Get(ref, remote.WithAuth(auth))
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
				"--image",
				"docker.io/feqlQoDIHc/bcfHFHHXYF",
			)).To(HaveOccurred())
		})

		It("should fail in case annotation is invalid", func() {
			tag := pushImage("test1")

			Expect(run(
				"--image",
				tag.String(),
				"--annotation",
				"org.opencontainers.image.url*https://my-company.com/images",
			)).To(HaveOccurred())
		})

		It("should fail in case label is invalid", func() {
			tag := pushImage("test2")

			Expect(run(
				"--image",
				tag.String(),
				"--label",
				" description*image description",
			)).To(HaveOccurred())
		})
	})

	Context("mutating the image", func() {
		It("should mutate an image with single annotation", func() {
			tag := pushImage("test3")

			withDockerConfigJSON(func(dockerConfigJSONPath string) {
				Expect(run(
					"--image",
					tag.String(),
					"--annotation",
					"org.opencontainers.image.url=https://my-company.com/images",
					"--secret-path",
					dockerConfigJSONPath,
				)).ToNot(HaveOccurred())
			})

			Expect(getImageAnnotation(tag.String(), "org.opencontainers.image.url")).
				To(Equal("https://my-company.com/images"))
		})

		It("should mutate an image with multiple annotations", func() {
			tag := pushImage("test4")

			withDockerConfigJSON(func(dockerConfigJSONPath string) {
				Expect(run(
					"--image",
					tag.String(),
					"--annotation",
					"org.opencontainers.image.url=https://my-company.com/images",
					"--annotation",
					"org.opencontainers.image.source=https://github.com/org/repo",
					"--secret-path",
					dockerConfigJSONPath,
				)).ToNot(HaveOccurred())
			})

			Expect(getImageAnnotation(tag.String(), "org.opencontainers.image.url")).
				To(Equal("https://my-company.com/images"))

			Expect(getImageAnnotation(tag.String(), "org.opencontainers.image.source")).
				To(Equal("https://github.com/org/repo"))
		})

		It("should mutate an image with single label", func() {
			tag := pushImage("test5")

			withDockerConfigJSON(func(dockerConfigJSONPath string) {
				Expect(run(
					"--image",
					tag.String(),
					"--label",
					"description=image description",
					"--secret-path",
					dockerConfigJSONPath,
				)).ToNot(HaveOccurred())
			})

			Expect(getImageConfigLabel(tag.String(), "description")).
				To(Equal("image description"))
		})

		It("should mutate an image with multiple labels", func() {
			tag := pushImage("test6")

			withDockerConfigJSON(func(dockerConfigJSONPath string) {
				Expect(run(
					"--image",
					tag.String(),
					"--label",
					"description=image description",
					"--label",
					"maintainer=team@my-company.com",
					"--secret-path",
					dockerConfigJSONPath,
				)).ToNot(HaveOccurred())
			})

			Expect(getImageConfigLabel(tag.String(), "description")).
				To(Equal("image description"))

			Expect(getImageConfigLabel(tag.String(), "maintainer")).
				To(Equal("team@my-company.com"))
		})

		It("should mutate an image with both annotation and label", func() {
			tag := pushImage("test7")

			withDockerConfigJSON(func(dockerConfigJSONPath string) {
				Expect(run(
					"--image",
					tag.String(),
					"--label",
					"description=image description",
					"--annotation",
					"org.opencontainers.image.url=https://my-company.com/images",
					"--secret-path",
					dockerConfigJSONPath,
				)).ToNot(HaveOccurred())
			})

			Expect(getImageConfigLabel(tag.String(), "description")).
				To(Equal("image description"))

			Expect(getImageAnnotation(tag.String(), "org.opencontainers.image.url")).
				To(Equal("https://my-company.com/images"))
		})
	})

	Context("store result after image mutation", func() {
		It("should store image digest into file specified in --result-file-image-digest flags", func() {
			tag := pushImage("test8")

			withTempFile("image-digest", func(filename string) {
				withDockerConfigJSON(func(dockerConfigJSONPath string) {
					Expect(run(
						"--image",
						tag.String(),
						"--annotation",
						"org.opencontainers.image.url=https://my-company.com/images",
						"--result-file-image-digest",
						filename,
						"--secret-path",
						dockerConfigJSONPath,
					)).ToNot(HaveOccurred())
				})

				Expect(filecontent(filename)).To(Equal(getImageDigest(tag).String()))
			})
		})

		It("should store image size into file specified in result-file-image-size flags", func() {
			tag := pushImage("test9")

			withTempFile("image-size", func(filename string) {
				withDockerConfigJSON(func(dockerConfigJSONPath string) {
					Expect(run(
						"--image",
						tag.String(),
						"--annotation",
						"org.opencontainers.image.url=https://my-company.com/images",
						"--result-file-image-size",
						filename,
						"--secret-path",
						dockerConfigJSONPath,
					)).ToNot(HaveOccurred())

					size := getCompressedImageSize(getImage(tag))
					Expect(filecontent(filename)).To(Equal(strconv.FormatInt(size, 10)))
				})
			})
		})
	})
})

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/shipwright-io/build/cmd/bundle"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/shipwright-io/build/pkg/bundle"
	"github.com/shipwright-io/build/pkg/image"
)

var _ = Describe("Bundle Loader", func() {
	const exampleImage = "ghcr.io/shipwright-io/sample-go/source-bundle:latest"

	run := func(args ...string) error {
		// discard log output
		log.SetOutput(io.Discard)

		// discard stderr output
		var tmp = os.Stderr
		os.Stderr = nil
		defer func() { os.Stderr = tmp }()

		os.Args = append([]string{"tool"}, args...)
		return Do(context.Background())
	}

	withTempDir := func(f func(target string)) {
		path, err := os.MkdirTemp(os.TempDir(), "bundle")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(path)

		f(path)
	}

	withTempFile := func(pattern string, f func(filename string)) {
		file, err := os.CreateTemp(os.TempDir(), pattern)
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove(file.Name())

		f(file.Name())
	}

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

	filecontent := func(path string) string {
		data, err := os.ReadFile(path)
		Expect(err).ToNot(HaveOccurred())
		return string(data)
	}

	getImage := func(tag name.Tag) containerreg.Image {
		ref, err := name.ParseReference(tag.String())
		Expect(err).To(BeNil())

		desc, err := remote.Get(ref)
		Expect(err).To(BeNil())

		img, err := desc.Image()
		Expect(err).To(BeNil())

		return img
	}

	getImageDigest := func(tag name.Tag) containerreg.Hash {
		digest, err := getImage(tag).Digest()
		Expect(err).To(BeNil())

		return digest
	}

	Context("validations and error cases", func() {
		It("should succeed in case the help is requested", func() {
			Expect(run("--help")).To(Succeed())
		})

		It("should fail in case the image is not specified", func() {
			Expect(run(
				"--image", "",
			)).To(HaveOccurred())
		})

		It("should fail in case the provided credentials do not match the required registry", func() {
			withTempFile("config.json", func(filename string) {
				Expect(os.WriteFile(filename, []byte(`{}`), 0644)).To(BeNil())
				Expect(run(
					"--image", "secret.typo.registry.com/foo:bar",
					"--secret-path", filename,
				)).To(MatchError("failed to find registry credentials for secret.typo.registry.com, available configurations: none"))

				Expect(os.WriteFile(filename, []byte(`{"auths":{"secret.private.registry.com":{"auth":"Zm9vQGJhci5jb206RGlkWW91UmVhbGx5RGVjb2RlVGhpcz8K"}}}`), 0644)).To(BeNil())
				Expect(run(
					"--image", "secret.typo.registry.com/foo:bar",
					"--secret-path", filename,
				)).To(MatchError("failed to find registry credentials for secret.typo.registry.com, available configurations: secret.private.registry.com"))
			})
		})
	})

	Context("Pulling image anonymously", func() {
		It("should pull and unbundle an image from a public registry", func() {
			withTempDir(func(target string) {
				Expect(run(
					"--image", exampleImage,
					"--target", target,
				)).To(Succeed())

				Expect(filepath.Join(target, "LICENSE")).To(BeAnExistingFile())
			})
		})

		It("should store image digest into file specified in --result-file-image-digest flags", func() {
			withTempDir(func(target string) {
				withTempFile("image-digest", func(filename string) {
					Expect(run(
						"--image", exampleImage,
						"--target", target,
						"--result-file-image-digest", filename,
					)).To(Succeed())

					tag, err := name.NewTag(exampleImage)
					Expect(err).ToNot(HaveOccurred())

					Expect(filecontent(filename)).To(Equal(getImageDigest(tag).String()))
				})
			})
		})
	})

	Context("Pulling image from private location", func() {
		var testImage string
		var dockerConfigFile string

		var copyImage = func(src, dst name.Reference) {
			options, _, err := image.GetOptions(context.TODO(), src, true, dockerConfigFile, "test-agent")
			Expect(err).ToNot(HaveOccurred())

			srcDesc, err := remote.Get(src, options...)
			Expect(err).ToNot(HaveOccurred())

			srcImage, err := srcDesc.Image()
			Expect(err).ToNot(HaveOccurred())

			options, _, err = image.GetOptions(context.TODO(), dst, true, dockerConfigFile, "test-agent")
			Expect(err).ToNot(HaveOccurred())

			err = remote.Write(dst, srcImage, options...)
			Expect(err).ToNot(HaveOccurred())
		}

		BeforeEach(func() {
			registryLocation, ok := os.LookupEnv("TEST_BUNDLE_REGISTRY_TARGET")
			if !ok {
				Skip("skipping test case with private registry location, because TEST_BUNDLE_REGISTRY_TARGET environment variable is not set, i.e. 'docker.io/some-namespace'")
			}

			dockerConfigFile, ok = os.LookupEnv("TEST_BUNDLE_DOCKERCONFIGFILE")
			if !ok {
				Skip("skipping test case with private registry, because TEST_BUNDLE_DOCKERCONFIGFILE environment variable is not set, i.e. '$HOME/.docker/config.json'")
			}

			testImage = fmt.Sprintf("%s/%s:%s",
				registryLocation,
				rand.String(5),
				"source",
			)

			src, err := name.ParseReference(exampleImage)
			Expect(err).ToNot(HaveOccurred())

			dst, err := name.ParseReference(testImage)
			Expect(err).ToNot(HaveOccurred())

			copyImage(src, dst)
		})

		AfterEach(func() {
			if testImage != "" {
				ref, err := name.ParseReference(testImage)
				Expect(err).ToNot(HaveOccurred())

				options, auth, err := image.GetOptions(context.TODO(), ref, true, dockerConfigFile, "test-agent")
				Expect(err).ToNot(HaveOccurred())

				// Delete test image (best effort)
				_ = image.Delete(ref, options, *auth)
			}
		})

		It("should pull and unpack an image from a private registry", func() {
			withTempDir(func(target string) {
				Expect(run(
					"--image", testImage,
					"--secret-path", dockerConfigFile,
					"--target", target,
				)).To(Succeed())

				Expect(filepath.Join(target, "LICENSE")).To(BeAnExistingFile())
			})
		})

		It("should delete the image after it was pulled", func() {
			withTempDir(func(target string) {
				Expect(run(
					"--image", testImage,
					"--prune",
					"--secret-path", dockerConfigFile,
					"--target", target,
				)).To(Succeed())

				Expect(filepath.Join(target, "LICENSE")).To(BeAnExistingFile())

				ref, err := name.ParseReference(testImage)
				Expect(err).ToNot(HaveOccurred())

				options, _, err := image.GetOptions(context.TODO(), ref, true, dockerConfigFile, "test-agent")
				Expect(err).ToNot(HaveOccurred())

				_, err = remote.Head(ref, options...)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("Result file checks", func() {
		tmpFile := func(dir string, name string, data []byte, timestamp time.Time) {
			var path = filepath.Join(dir, name)

			Expect(os.WriteFile(
				path,
				data,
				os.FileMode(0644),
			)).To(Succeed())

			Expect(os.Chtimes(
				path,
				timestamp,
				timestamp,
			)).To(Succeed())
		}

		// Creates a controlled reference image with one file called "file" with modification
		// timestamp of Friday, February 13, 2009 11:31:30 PM (unix timestamp 1234567890)
		withReferenceImage := func(f func(dig name.Digest)) {
			withTempRegistry(func(endpoint string) {
				withTempDir(func(target string) {
					timestamp := time.Unix(1234567890, 0)

					ref, err := name.ParseReference(fmt.Sprintf("%s/namespace/image:tag", endpoint))
					Expect(err).ToNot(HaveOccurred())
					Expect(ref).ToNot(BeNil())

					tmpFile(target, "file", []byte("foobar"), timestamp)

					dig, err := bundle.PackAndPush(ref, target)
					Expect(err).ToNot(HaveOccurred())
					Expect(dig).ToNot(BeNil())

					f(dig)
				})
			})
		}

		It("should store source timestamp in result file", func() {
			withTempDir(func(target string) {
				withTempDir(func(result string) {
					withReferenceImage(func(dig name.Digest) {
						resultSourceTimestamp := filepath.Join(result, "source-timestamp")

						Expect(run(
							"--image", dig.String(),
							"--target", target,
							"--result-file-source-timestamp", resultSourceTimestamp,
						)).To(Succeed())

						Expect(filecontent(resultSourceTimestamp)).To(Equal("1234567890"))
					})
				})
			})
		})
	})
})

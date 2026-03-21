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
	"path"
	"strconv"
	"strings"
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
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/rand"
)

var _ = Describe("Image Processing Resource", Ordered, func() {
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
		// #nosec G304 ok in tests
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

	Context("Vulnerability Scanning", func() {
		directory := path.Join("..", "..", "test", "data", "images", "vuln-image-in-oci")

		It("should run vulnerability scanning if it is enabled and output vulnerabilities equal to the limit defined", func() {
			vulnOptions := &buildapi.VulnerabilityScanOptions{
				Enabled: true,
			}
			withTempRegistry(func(endpoint string) {
				tag, err := name.NewTag(fmt.Sprintf("%s/%s:%s", endpoint, "temp-image", rand.String(5)))
				Expect(err).ToNot(HaveOccurred())
				vulnSettings := &resources.VulnerablilityScanParams{VulnerabilityScanOptions: *vulnOptions}
				withTempFile("vuln-scan-result", func(filename string) {
					Expect(run(
						"--insecure",
						"--image", tag.String(),
						"--push", directory,
						"--vuln-settings", vulnSettings.String(),
						"--result-file-image-vulnerabilities", filename,
						"--vuln-count-limit", "10",
					)).ToNot(HaveOccurred())
					output := filecontent(filename)
					Expect(output).To(ContainSubstring("CVE-2019-8457"))
					vulnerabilities := strings.Split(output, ",")
					Expect(vulnerabilities).To(HaveLen(10))
				})
			})
		})

		It("should push the image if vulnerabilities are found and fail is false", func() {
			vulnOptions := &buildapi.VulnerabilityScanOptions{
				Enabled:       true,
				FailOnFinding: false,
			}
			withTempRegistry(func(endpoint string) {
				tag, err := name.NewTag(fmt.Sprintf("%s/%s:%s", endpoint, "temp-image", rand.String(5)))
				Expect(err).ToNot(HaveOccurred())
				vulnSettings := &resources.VulnerablilityScanParams{VulnerabilityScanOptions: *vulnOptions}
				withTempFile("vuln-scan-result", func(filename string) {
					Expect(run(
						"--insecure",
						"--image", tag.String(),
						"--push", directory,
						"--vuln-settings", vulnSettings.String(),
						"--result-file-image-vulnerabilities", filename,
					)).ToNot(HaveOccurred())
					output := filecontent(filename)
					Expect(output).To(ContainSubstring("CVE-2019-8457"))
				})

				ref, err := name.ParseReference(tag.String())
				Expect(err).ToNot(HaveOccurred())
				_, err = remote.Get(ref)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		It("should not push the image if vulnerabilities are found and fail is true", func() {
			vulnOptions := &buildapi.VulnerabilityScanOptions{
				Enabled:       true,
				FailOnFinding: true,
			}
			withTempRegistry(func(endpoint string) {
				tag, err := name.NewTag(fmt.Sprintf("%s/%s:%s", endpoint, "temp-image", rand.String(5)))
				Expect(err).ToNot(HaveOccurred())
				vulnSettings := &resources.VulnerablilityScanParams{VulnerabilityScanOptions: *vulnOptions}
				withTempFile("vuln-scan-result", func(filename string) {
					Expect(run(
						"--insecure",
						"--image", tag.String(),
						"--push", directory,
						"--vuln-settings", vulnSettings.String(),
						"--result-file-image-vulnerabilities", filename,
					)).To(HaveOccurred())
					output := filecontent(filename)
					Expect(output).To(ContainSubstring("CVE-2019-8457"))
				})

				ref, err := name.ParseReference(tag.String())
				Expect(err).ToNot(HaveOccurred())

				_, err = remote.Get(ref)
				Expect(err).To(HaveOccurred())
			})
		})

		It("should run vulnerability scanning on an image that is already pushed by the strategy", func() {
			ignoreVulnerabilities := buildapi.IgnoredHigh
			vulnOptions := &buildapi.VulnerabilityScanOptions{
				Enabled:       true,
				FailOnFinding: true,
				Ignore: &buildapi.VulnerabilityIgnoreOptions{
					Severity: &ignoreVulnerabilities,
				},
			}

			withTempRegistry(func(endpoint string) {
				originalImageRef := "ghcr.io/shipwright-io/shipwright-samples/node:12"
				srcRef, err := name.ParseReference(originalImageRef)
				Expect(err).ToNot(HaveOccurred())

				// Pull the original image
				originalImage, err := remote.Image(srcRef)
				Expect(err).ToNot(HaveOccurred())

				// Tag the image with a new name
				tag, err := name.NewTag(fmt.Sprintf("%s/%s:%s", endpoint, "temp-image", rand.String(5)))
				Expect(err).ToNot(HaveOccurred())

				err = remote.Write(tag, originalImage)
				Expect(err).ToNot(HaveOccurred())

				vulnSettings := &resources.VulnerablilityScanParams{VulnerabilityScanOptions: *vulnOptions}
				withTempFile("vuln-scan-result", func(filename string) {
					Expect(run(
						"--insecure",
						"--image", tag.String(),
						"--vuln-settings", vulnSettings.String(),
						"--result-file-image-vulnerabilities", filename,
					)).ToNot(HaveOccurred())
					output := filecontent(filename)
					Expect(output).To(ContainSubstring("CVE-2019-12900"))
				})

				ref, err := name.ParseReference(tag.String())
				Expect(err).ToNot(HaveOccurred())

				_, err = remote.Get(ref)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})

var _ = Describe("Source Bundle Push", func() {
	run := func(args ...string) error {
		log.SetOutput(GinkgoWriter)
		os.Args = append([]string{"tool"}, args...)
		tmp := os.Stderr
		defer func() { os.Stderr = tmp }()
		os.Stderr = nil
		return Execute(context.Background())
	}

	AfterEach(func() {
		pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	})

	It("should bundle a source directory and push it as an OCI artifact", func() {
		logLogger := log.Logger{}
		logLogger.SetOutput(GinkgoWriter)
		s := httptest.NewServer(registry.New(registry.Logger(&logLogger)))
		defer s.Close()
		u, err := url.Parse(s.URL)
		Expect(err).ToNot(HaveOccurred())
		endpoint := u.Host

		srcDir, err := os.MkdirTemp("", "source-bundle-test")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(srcDir)

		Expect(os.WriteFile(fmt.Sprintf("%s/main.go", srcDir), []byte("package main"), 0600)).To(Succeed())
		Expect(os.MkdirAll(fmt.Sprintf("%s/pkg", srcDir), 0750)).To(Succeed())
		Expect(os.WriteFile(fmt.Sprintf("%s/pkg/lib.go", srcDir), []byte("package pkg"), 0600)).To(Succeed())

		imageRef := fmt.Sprintf("%s/test/source-bundle:latest", endpoint)
		digestFile, err := os.CreateTemp("", "digest")
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove(digestFile.Name())

		Expect(run(
			"--push-source-bundle", srcDir,
			"--source-bundle-image", imageRef,
			"--insecure",
			"--result-file-image-digest", digestFile.Name(),
		)).To(Succeed())

		ref, err := name.ParseReference(imageRef)
		Expect(err).ToNot(HaveOccurred())

		desc, err := remote.Get(ref)
		Expect(err).ToNot(HaveOccurred())

		img, err := desc.Image()
		Expect(err).ToNot(HaveOccurred())

		layers, err := img.Layers()
		Expect(err).ToNot(HaveOccurred())
		Expect(layers).To(HaveLen(1))

		digestData, err := os.ReadFile(digestFile.Name())
		Expect(err).ToNot(HaveOccurred())
		Expect(string(digestData)).To(HavePrefix("sha256:"))
	})

	It("should fail when --source-bundle-image is missing", func() {
		srcDir, err := os.MkdirTemp("", "source-bundle-test")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(srcDir)

		Expect(run(
			"--push-source-bundle", srcDir,
		)).To(HaveOccurred())
	})
})

var _ = Describe("Assemble Index", func() {
	run := func(args ...string) error {
		log.SetOutput(GinkgoWriter)
		os.Args = append([]string{"tool"}, args...)
		tmp := os.Stderr
		defer func() { os.Stderr = tmp }()
		os.Stderr = nil
		return Execute(context.Background())
	}

	AfterEach(func() {
		pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	})

	It("should assemble an OCI image index from per-platform images", func() {
		logLogger := log.Logger{}
		logLogger.SetOutput(GinkgoWriter)
		s := httptest.NewServer(registry.New(registry.Logger(&logLogger)))
		defer s.Close()
		u, err := url.Parse(s.URL)
		Expect(err).ToNot(HaveOccurred())
		endpoint := u.Host

		amd64Tag := fmt.Sprintf("%s/test/app-linux-amd64:latest", endpoint)
		arm64Tag := fmt.Sprintf("%s/test/app-linux-arm64:latest", endpoint)

		amd64Ref, err := name.ParseReference(amd64Tag)
		Expect(err).ToNot(HaveOccurred())
		arm64Ref, err := name.ParseReference(arm64Tag)
		Expect(err).ToNot(HaveOccurred())

		Expect(remote.Write(amd64Ref, empty.Image)).To(Succeed())
		Expect(remote.Write(arm64Ref, empty.Image)).To(Succeed())

		amd64Digest, err := empty.Image.Digest()
		Expect(err).ToNot(HaveOccurred())
		arm64Digest, err := empty.Image.Digest()
		Expect(err).ToNot(HaveOccurred())

		outputRef := fmt.Sprintf("%s/test/app:latest", endpoint)
		digestFile, err := os.CreateTemp("", "index-digest")
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove(digestFile.Name())

		Expect(run(
			"--assemble-index",
			"--image", outputRef,
			"--insecure",
			"--platform-image", fmt.Sprintf("linux/amd64=%s@%s", amd64Tag, amd64Digest),
			"--platform-image", fmt.Sprintf("linux/arm64=%s@%s", arm64Tag, arm64Digest),
			"--result-file-image-digest", digestFile.Name(),
		)).To(Succeed())

		digestData, err := os.ReadFile(digestFile.Name())
		Expect(err).ToNot(HaveOccurred())
		Expect(string(digestData)).To(HavePrefix("sha256:"))

		ref, err := name.ParseReference(outputRef)
		Expect(err).ToNot(HaveOccurred())

		desc, err := remote.Get(ref)
		Expect(err).ToNot(HaveOccurred())
		idx, err := desc.ImageIndex()
		Expect(err).ToNot(HaveOccurred())

		indexManifest, err := idx.IndexManifest()
		Expect(err).ToNot(HaveOccurred())
		Expect(indexManifest.Manifests).To(HaveLen(2))
	})

	It("should fail when --image is missing", func() {
		Expect(run(
			"--assemble-index",
			"--platform-image", "linux/amd64=localhost:5000/test@sha256:abc",
		)).To(HaveOccurred())
	})

	It("should fail when no --platform-image is provided", func() {
		Expect(run(
			"--assemble-index",
			"--image", "localhost:5000/test:latest",
		)).To(HaveOccurred())
	})
})

var _ = Describe("ParsePlatformImages", func() {
	It("should parse valid platform image entries", func() {
		entries, err := ParsePlatformImages([]string{
			"linux/amd64=registry.example.com/app@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"linux/arm64=registry.example.com/app@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(entries).To(HaveLen(2))
		Expect(entries[0].OS).To(Equal("linux"))
		Expect(entries[0].Arch).To(Equal("amd64"))
		Expect(entries[1].OS).To(Equal("linux"))
		Expect(entries[1].Arch).To(Equal("arm64"))
	})

	It("should fail when the format is invalid", func() {
		_, err := ParsePlatformImages([]string{"not-a-valid-entry"})
		Expect(err).To(HaveOccurred())
	})
})

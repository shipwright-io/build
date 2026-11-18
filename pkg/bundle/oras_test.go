// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	. "github.com/shipwright-io/build/pkg/bundle"
)

// blobLayer is a layer whose blob is stored as given, the way ORAS stores files.
type blobLayer struct {
	compressed   []byte
	uncompressed []byte
	mediaType    types.MediaType
}

func (l *blobLayer) Digest() (containerreg.Hash, error) {
	h, _, err := containerreg.SHA256(bytes.NewReader(l.compressed))
	return h, err
}

func (l *blobLayer) DiffID() (containerreg.Hash, error) {
	h, _, err := containerreg.SHA256(bytes.NewReader(l.uncompressed))
	return h, err
}

func (l *blobLayer) Compressed() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(l.compressed)), nil
}

func (l *blobLayer) Uncompressed() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(l.uncompressed)), nil
}

func (l *blobLayer) Size() (int64, error) { return int64(len(l.compressed)), nil }

func (l *blobLayer) MediaType() (types.MediaType, error) { return l.mediaType, nil }

func fileLayer(content string) *blobLayer {
	return &blobLayer{compressed: []byte(content), uncompressed: []byte(content), mediaType: "text/plain"}
}

// directoryLayer packs the files into a gzipped tarball below dir, like oras push does for a directory.
func directoryLayer(dir string, modTime time.Time, files map[string]string) *blobLayer {
	return directoryLayerWithMode(dir, modTime, 0644, files)
}

func directoryLayerWithMode(dir string, modTime time.Time, mode int64, files map[string]string) *blobLayer {
	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	Expect(tw.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: dir + "/", Mode: 0755, ModTime: modTime})).To(Succeed())
	for fileName, content := range files {
		Expect(tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     dir + "/" + fileName,
			Mode:     mode,
			Size:     int64(len(content)),
			ModTime:  modTime,
		})).To(Succeed())
		_, err := tw.Write([]byte(content))
		Expect(err).ToNot(HaveOccurred())
	}
	Expect(tw.Close()).To(Succeed())

	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	_, err := gw.Write(tarBuf.Bytes())
	Expect(err).ToNot(HaveOccurred())
	Expect(gw.Close()).To(Succeed())

	return &blobLayer{compressed: gzBuf.Bytes(), uncompressed: tarBuf.Bytes(), mediaType: types.OCILayer}
}

func orasArtifact(created string, addenda ...mutate.Addendum) containerreg.Image {
	image, err := mutate.Append(mutate.MediaType(empty.Image, types.OCIManifestSchema1), addenda...)
	Expect(err).ToNot(HaveOccurred())

	if created != "" {
		image = mutate.Annotations(image, map[string]string{ocispec.AnnotationCreated: created}).(containerreg.Image)
	}

	return image
}

var _ = Describe("Unpacking ORAS artifacts", func() {
	var withTempDir = func(f func(target string)) {
		tempDir, err := os.MkdirTemp("", "bundle-oras")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(tempDir)

		f(tempDir)
	}

	var withTempRegistry = func(f func(endpoint string)) {
		logLogger := log.Logger{}
		logLogger.SetOutput(GinkgoWriter)

		s := httptest.NewServer(registry.New(registry.Logger(&logLogger)))
		defer s.Close()

		u, err := url.Parse(s.URL)
		Expect(err).ToNot(HaveOccurred())

		f(u.Host)
	}

	dirModTime := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	It("should write file layers as is and unpack directory layers", func() {
		image := orasArtifact("",
			mutate.Addendum{
				Layer:       fileLayer("hello world"),
				Annotations: map[string]string{ocispec.AnnotationTitle: "hello.txt"},
			},
			mutate.Addendum{
				Layer: directoryLayer("src", dirModTime, map[string]string{"main.go": "package main"}),
				Annotations: map[string]string{
					ocispec.AnnotationTitle:       "src",
					"io.deis.oras.content.unpack": "true",
				},
			},
		)

		withTempDir(func(target string) {
			details, err := UnpackImage(image, target)
			Expect(err).ToNot(HaveOccurred())

			Expect(os.ReadFile(filepath.Join(target, "hello.txt"))).To(Equal([]byte("hello world")))       // #nosec G304 test paths
			Expect(os.ReadFile(filepath.Join(target, "src", "main.go"))).To(Equal([]byte("package main"))) // #nosec G304 test paths
			Expect(details.MostRecentFileTimestamp).ToNot(BeNil())
			Expect(details.MostRecentFileTimestamp.Equal(dirModTime)).To(BeTrue())
		})
	})

	It("should use the creation time of the artifact when only files are stored", func() {
		image := orasArtifact("2026-01-02T03:04:05Z", mutate.Addendum{
			Layer:       fileLayer("hello world"),
			Annotations: map[string]string{ocispec.AnnotationTitle: "hello.txt"},
		})

		withTempDir(func(target string) {
			details, err := UnpackImage(image, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(details.MostRecentFileTimestamp).ToNot(BeNil())
			Expect(details.MostRecentFileTimestamp.Equal(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))).To(BeTrue())
		})
	})

	DescribeTable("should reject a file title that escapes the target directory",
		func(title string) {
			image := orasArtifact("", mutate.Addendum{
				Layer:       fileLayer("owned"),
				Annotations: map[string]string{ocispec.AnnotationTitle: title},
			})

			withTempDir(func(target string) {
				_, err := UnpackImage(image, filepath.Join(target, "target"))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("escapes the target directory"))
				Expect(filepath.Join(target, "canary")).ToNot(BeAnExistingFile())
			})
		},
		Entry("parent directory", "../canary"),
		Entry("target directory itself", "."),
	)

	DescribeTable("should not follow a symlink in the target directory that points outside of it",
		func(title string, wantErr bool) {
			if runtime.GOOS == "windows" {
				Skip("creating symlinks needs extra privileges on Windows")
			}

			image := orasArtifact("", mutate.Addendum{
				Layer:       fileLayer("owned"),
				Annotations: map[string]string{ocispec.AnnotationTitle: title},
			})

			withTempDir(func(tempDir string) {
				outside := filepath.Join(tempDir, "outside")
				Expect(os.Mkdir(outside, 0o750)).To(Succeed())
				canary := filepath.Join(outside, "canary")
				Expect(os.WriteFile(canary, []byte("untouched"), 0o600)).To(Succeed())

				target := filepath.Join(tempDir, "target")
				Expect(os.Mkdir(target, 0o750)).To(Succeed())
				Expect(os.Symlink(outside, filepath.Join(target, "link"))).To(Succeed())
				Expect(os.Symlink(canary, filepath.Join(target, "file"))).To(Succeed())

				_, err := UnpackImage(image, target)
				Expect(os.ReadFile(canary)).To(Equal([]byte("untouched"))) // #nosec G304 test paths

				if wantErr {
					Expect(err).To(HaveOccurred())
					return
				}

				// A symlink at the file name is replaced, it is not written through
				Expect(err).ToNot(HaveOccurred())
				info, err := os.Lstat(filepath.Join(target, title))
				Expect(err).ToNot(HaveOccurred())
				Expect(info.Mode().IsRegular()).To(BeTrue())
				Expect(os.ReadFile(filepath.Join(target, title))).To(Equal([]byte("owned")))
			})
		},
		Entry("parent directory symlink", "link/canary", true),
		Entry("final file name symlink", "file", false),
	)

	It("should replace files of earlier layers that overlap with later ones", func() {
		image := orasArtifact("",
			mutate.Addendum{
				Layer: directoryLayer("src", dirModTime, map[string]string{"main.go": "old-content"}),
				Annotations: map[string]string{
					ocispec.AnnotationTitle:       "src",
					"io.deis.oras.content.unpack": "true",
				},
			},
			mutate.Addendum{
				Layer: directoryLayerWithMode("src", dirModTime, 0755, map[string]string{"main.go": "new"}),
				Annotations: map[string]string{
					ocispec.AnnotationTitle:       "src",
					"io.deis.oras.content.unpack": "true",
				},
			},
		)

		withTempDir(func(target string) {
			_, err := UnpackImage(image, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(os.ReadFile(filepath.Join(target, "src", "main.go"))).To(Equal([]byte("new")))

			if runtime.GOOS != "windows" {
				info, err := os.Stat(filepath.Join(target, "src", "main.go"))
				Expect(err).ToNot(HaveOccurred())
				Expect(info.Mode().Perm()).To(Equal(os.FileMode(0755)))
			}
		})
	})

	It("should replace read-only files of earlier layers", func() {
		layerAnnotations := map[string]string{
			ocispec.AnnotationTitle:       "src",
			"io.deis.oras.content.unpack": "true",
		}
		image := orasArtifact("",
			mutate.Addendum{
				Layer:       directoryLayerWithMode("src", dirModTime, 0444, map[string]string{"main.go": "old-content"}),
				Annotations: layerAnnotations,
			},
			mutate.Addendum{
				Layer:       directoryLayerWithMode("src", dirModTime, 0444, map[string]string{"main.go": "new"}),
				Annotations: layerAnnotations,
			},
			mutate.Addendum{
				Layer:       fileLayer("file-content"),
				Annotations: map[string]string{ocispec.AnnotationTitle: "src/main.go"},
			},
		)

		withTempDir(func(target string) {
			_, err := UnpackImage(image, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(os.ReadFile(filepath.Join(target, "src", "main.go"))).To(Equal([]byte("file-content")))
		})
	})

	It("should pull and unpack an ORAS artifact from a registry", func() {
		withTempRegistry(func(endpoint string) {
			ref, err := name.ParseReference(fmt.Sprintf("%s/namespace/unit-test-pkg-bundle-oras-%s:latest", endpoint, rand.String(5)))
			Expect(err).ToNot(HaveOccurred())

			image := orasArtifact("", mutate.Addendum{
				Layer:       fileLayer("hello world"),
				Annotations: map[string]string{ocispec.AnnotationTitle: "hello.txt"},
			})
			Expect(remote.Write(ref, image)).To(Succeed())

			withTempDir(func(target string) {
				_, err := PullAndUnpack(ref, target)
				Expect(err).ToNot(HaveOccurred())
				Expect(os.ReadFile(filepath.Join(target, "hello.txt"))).To(Equal([]byte("hello world")))
			})
		})
	})
})

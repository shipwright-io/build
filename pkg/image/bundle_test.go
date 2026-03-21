// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/shipwright-io/build/pkg/image"
)

var _ = Describe("BundleSourceDirectory", func() {

	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "bundle-test-")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() {
			_ = os.RemoveAll(tmpDir)
		})
	})

	It("creates a valid OCI image from a directory with files", func() {
		Expect(os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n"), 0600)).To(Succeed())
		Expect(os.Mkdir(filepath.Join(tmpDir, "pkg"), 0750)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(tmpDir, "pkg", "lib.go"), []byte("package pkg\n"), 0600)).To(Succeed())

		img, err := image.BundleSourceDirectory(tmpDir)
		Expect(err).ToNot(HaveOccurred())
		Expect(img).ToNot(BeNil())

		layers, err := img.Layers()
		Expect(err).ToNot(HaveOccurred())
		Expect(layers).To(HaveLen(1))

		digest, err := img.Digest()
		Expect(err).ToNot(HaveOccurred())
		Expect(digest.String()).ToNot(BeEmpty())
	})

	It("produces a layer whose compressed content is readable after the function returns", func() {
		Expect(os.WriteFile(filepath.Join(tmpDir, "hello.txt"), []byte("hello world\n"), 0600)).To(Succeed())

		img, err := image.BundleSourceDirectory(tmpDir)
		Expect(err).ToNot(HaveOccurred())

		layers, err := img.Layers()
		Expect(err).ToNot(HaveOccurred())
		Expect(layers).To(HaveLen(1))

		rc, err := layers[0].Compressed()
		Expect(err).ToNot(HaveOccurred())
		defer rc.Close()

		gzReader, err := gzip.NewReader(rc)
		Expect(err).ToNot(HaveOccurred())
		defer gzReader.Close()

		tarReader := tar.NewReader(gzReader)
		var found bool
		for {
			hdr, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			Expect(err).ToNot(HaveOccurred())
			if hdr.Name == "hello.txt" {
				found = true
				content, err := io.ReadAll(tarReader)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal("hello world\n"))
			}
		}
		Expect(found).To(BeTrue(), "expected hello.txt in the tar layer")
	})

	It("dereferences symlinks so the tar only contains regular files", func() {
		// Create a regular file and a symlink pointing to it
		Expect(os.WriteFile(filepath.Join(tmpDir, "target.txt"), []byte("symlink target\n"), 0600)).To(Succeed())
		Expect(os.Symlink(filepath.Join(tmpDir, "target.txt"), filepath.Join(tmpDir, "link.txt"))).To(Succeed())

		img, err := image.BundleSourceDirectory(tmpDir)
		Expect(err).ToNot(HaveOccurred())

		layers, err := img.Layers()
		Expect(err).ToNot(HaveOccurred())
		Expect(layers).To(HaveLen(1))

		rc, err := layers[0].Compressed()
		Expect(err).ToNot(HaveOccurred())
		defer rc.Close()

		gzReader, err := gzip.NewReader(rc)
		Expect(err).ToNot(HaveOccurred())
		defer gzReader.Close()

		tarReader := tar.NewReader(gzReader)
		var foundLink bool
		for {
			hdr, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			Expect(err).ToNot(HaveOccurred())
			if hdr.Name == "link.txt" {
				foundLink = true
				Expect(hdr.Typeflag).To(Equal(byte(tar.TypeReg)), "symlink should be dereferenced to a regular file")
				Expect(hdr.Linkname).To(BeEmpty(), "dereferenced entry should have no Linkname")
				content, err := io.ReadAll(tarReader)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal("symlink target\n"))
			}
		}
		Expect(foundLink).To(BeTrue(), "expected link.txt in the tar layer")
	})

	It("returns an error for a non-existent directory", func() {
		_, err := image.BundleSourceDirectory("/nonexistent/path")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("source directory does not exist"))
	})
})

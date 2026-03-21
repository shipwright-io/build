// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/shipwright-io/build/pkg/image"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AssembleImageIndex", func() {

	var registryHost string

	BeforeEach(func() {
		logger := log.New(io.Discard, "", 0)
		reg := registry.New(registry.Logger(logger))
		server := httptest.NewServer(reg)
		DeferCleanup(func() {
			server.Close()
		})
		registryHost = strings.ReplaceAll(server.URL, "http://", "")
	})

	It("assembles an image index from two platform images", func() {
		amd64Ref, err := name.ParseReference(fmt.Sprintf("%s/test/app-linux-amd64:latest", registryHost))
		Expect(err).ToNot(HaveOccurred())
		arm64Ref, err := name.ParseReference(fmt.Sprintf("%s/test/app-linux-arm64:latest", registryHost))
		Expect(err).ToNot(HaveOccurred())

		amd64Img, err := random.Image(256, 1)
		Expect(err).ToNot(HaveOccurred())
		Expect(remote.Write(amd64Ref, amd64Img)).To(Succeed())

		arm64Img, err := random.Image(256, 1)
		Expect(err).ToNot(HaveOccurred())
		Expect(remote.Write(arm64Ref, arm64Img)).To(Succeed())

		entries := []image.PlatformImageEntry{
			{OS: "linux", Arch: "amd64", ImageRef: amd64Ref},
			{OS: "linux", Arch: "arm64", ImageRef: arm64Ref},
		}

		idx, err := image.AssembleImageIndex(entries, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(idx).ToNot(BeNil())

		indexManifest, err := idx.IndexManifest()
		Expect(err).ToNot(HaveOccurred())
		Expect(indexManifest.Manifests).To(HaveLen(2))

		Expect(indexManifest.Manifests[0].Platform.OS).To(Equal("linux"))
		Expect(indexManifest.Manifests[0].Platform.Architecture).To(Equal("amd64"))
		Expect(indexManifest.Manifests[1].Platform.OS).To(Equal("linux"))
		Expect(indexManifest.Manifests[1].Platform.Architecture).To(Equal("arm64"))
	})

	It("returns an error with empty entries", func() {
		_, err := image.AssembleImageIndex(nil, nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("at least one platform image entry is required"))
	})

	It("returns an error when a platform image does not exist", func() {
		ref, err := name.ParseReference(fmt.Sprintf("%s/test/nonexistent:latest", registryHost))
		Expect(err).ToNot(HaveOccurred())

		entries := []image.PlatformImageEntry{
			{OS: "linux", Arch: "amd64", ImageRef: ref},
		}

		_, err = image.AssembleImageIndex(entries, nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("pulling image"))
	})
})

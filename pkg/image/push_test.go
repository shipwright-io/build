// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	"fmt"
	"io"
	"log"
	"net/http"
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

var _ = Describe("PushImageOrImageIndex", func() {

	var registryHost string

	BeforeEach(func() {
		logger := log.New(io.Discard, "", 0)
		reg := registry.New(registry.Logger(logger))
		// Use the following instead to see which requests happened
		// reg := registry.New()
		server := httptest.NewServer(reg)
		DeferCleanup(func() {
			server.Close()
		})
		registryHost = strings.ReplaceAll(server.URL, "http://", "")
	})

	Context("For an image", func() {

		img, err := random.Image(3245, 1)
		Expect(err).ToNot(HaveOccurred())

		It("pushes the image", func() {
			imageName, err := name.ParseReference(fmt.Sprintf("%s/%s/%s", registryHost, "test-namespace", "test-image"))
			Expect(err).ToNot(HaveOccurred())

			digest, size, err := image.PushImageOrImageIndex(imageName, img, nil, []remote.Option{})
			Expect(err).ToNot(HaveOccurred())
			Expect(digest).To(HavePrefix("sha"))
			// The size includes the manifest which depends on the config which depends on the registry host
			Expect(size > 3500).To(BeTrue())

			// Verify the existence of the manifest
			request, err := http.NewRequest("GET", fmt.Sprintf("http://%s/v2/test-namespace/test-image/manifests/latest", registryHost), nil)
			Expect(err).ToNot(HaveOccurred())
			response, err := http.DefaultClient.Do(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.StatusCode).To(Equal(200))
		})
	})

	Context("For an index", func() {

		index, err := random.Index(1234, 1, 2)
		Expect(err).ToNot(HaveOccurred())

		It("pushes the index", func() {
			imageName, err := name.ParseReference(fmt.Sprintf("%s/%s/%s:%s", registryHost, "test-namespace", "test-index", "test-tag"))
			Expect(err).ToNot(HaveOccurred())

			digest, size, err := image.PushImageOrImageIndex(imageName, nil, index, []remote.Option{})
			Expect(err).ToNot(HaveOccurred())
			Expect(digest).To(HavePrefix("sha"))
			// The size includes the manifest which depends on the config which depends on the registry host
			Expect(size).To(BeEquivalentTo(-1))

			// Verify the existence of the manifest
			request, err := http.NewRequest("GET", fmt.Sprintf("http://%s/v2/test-namespace/test-index/manifests/test-tag", registryHost), nil)
			Expect(err).ToNot(HaveOccurred())
			response, err := http.DefaultClient.Do(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.StatusCode).To(Equal(200))
		})
	})
})

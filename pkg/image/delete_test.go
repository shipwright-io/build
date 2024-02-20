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

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/shipwright-io/build/pkg/image"
	utils "github.com/shipwright-io/build/test/utils/v1beta1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Delete", func() {

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

	Context("For an image in a registry", func() {

		var imageName name.Reference

		BeforeEach(func() {
			img, err := random.Image(3245, 1)
			Expect(err).ToNot(HaveOccurred())

			imageName, err = name.ParseReference(fmt.Sprintf("%s/%s/%s", registryHost, "test-namespace", "test-image"))
			Expect(err).ToNot(HaveOccurred())

			_, _, err = image.PushImageOrImageIndex(imageName, img, nil, []remote.Option{})
			Expect(err).ToNot(HaveOccurred())

			// Verify the existence of the manifest
			Expect(fmt.Sprintf("http://%s/v2/test-namespace/test-image/manifests/latest", registryHost)).To(utils.Return(200))
		})

		It("deletes the image", func() {
			err := image.Delete(imageName, []remote.Option{}, authn.AuthConfig{})
			Expect(err).ToNot(HaveOccurred())

			// Verify the non-existence of the manifest
			Expect(fmt.Sprintf("http://%s/v2/test-namespace/test-image/manifests/latest", registryHost)).ToNot(utils.Return(200))
		})
	})
})

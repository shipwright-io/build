// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http/httptest"
	"strings"

	"github.com/google/go-containerregistry/pkg/registry"
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/shipwright-io/build/pkg/image"
)

var _ = Describe("Insecure registries", func() {

	var img containerreg.Image

	BeforeEach(func() {
		var err error
		img, err = random.Image(1024, 1)
		Expect(err).ToNot(HaveOccurred())
	})

	// The registry listens on 127.0.0.2, which is a loopback address that does not match the
	// localhost heuristics of go-containerregistry. The scheme is therefore only driven by
	// the insecure flag and not by the address of the registry.
	startRegistry := func(useTLS bool) string {
		logger := log.New(io.Discard, "", 0)

		server := httptest.NewUnstartedServer(registry.New(registry.Logger(logger)))
		server.Config.ErrorLog = logger
		Expect(server.Listener.Close()).To(Succeed())

		listener, err := net.Listen("tcp", "127.0.0.2:0")
		Expect(err).ToNot(HaveOccurred())
		server.Listener = listener

		if useTLS {
			server.StartTLS()
		} else {
			server.Start()
		}
		DeferCleanup(server.Close)

		return strings.NewReplacer("http://", "", "https://", "").Replace(server.URL)
	}

	DescribeTable("pushing an image",
		func(useTLS bool, insecure bool, expectSuccess bool) {
			registryHost := startRegistry(useTLS)

			imageName, err := image.ParseReference(fmt.Sprintf("%s/%s/%s", registryHost, "test-namespace", "test-image"), insecure)
			Expect(err).ToNot(HaveOccurred())

			options, _, err := image.GetOptions(context.TODO(), imageName, insecure, "", "test-agent")
			Expect(err).ToNot(HaveOccurred())

			_, _, err = image.PushImageOrImageIndex(imageName, img, nil, options)
			if expectSuccess {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
		},
		Entry("works for an HTTP registry when insecure is set", false, true, true),
		Entry("works for an HTTPS registry with a self-signed certificate when insecure is set", true, true, true),
		Entry("fails for an HTTP registry when insecure is not set", false, false, false),
		Entry("fails for an HTTPS registry with a self-signed certificate when insecure is not set", true, false, false),
	)
})

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	"github.com/shipwright-io/build/pkg/image"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Endpoints", func() {

	DescribeTable("the extraction of hostname and port",
		func(url string, expectedHost string, expectedPort int, expectError bool) {
			host, port, err := image.ExtractHostnamePort(url)
			if expectError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(host).To(Equal(expectedHost), "for "+url)
				Expect(port).To(Equal(expectedPort), "for "+url)
			}
		},
		Entry("Check a URL with default port", "registry.access.redhat.com/ubi9/ubi-minimal", "registry.access.redhat.com", 443, false),
		Entry("Check a URL with custom port", "registry.access.redhat.com:9443/ubi9/ubi-minimal", "registry.access.redhat.com", 9443, false),
		Entry("Check a URL without host", "ubuntu", "index.docker.io", 443, false),
		Entry("Check invalid URL", "ftp://registry.access.redhat.com/ubi9/ubi-minimal", "", 0, true),
	)
})

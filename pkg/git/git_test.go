// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package git_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/shipwright-io/build/pkg/git"
)

var _ = Describe("Git", func() {

	DescribeTable("the extraction of hostname and port",
		func(url string, expectedHost string, expectedPort int, expectError bool) {
			host, port, err := git.ExtractHostnamePort(url)
			if expectError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(host).To(Equal(expectedHost), "for "+url)
				Expect(port).To(Equal(expectedPort), "for "+url)
			}
		},
		Entry("Check heritage SSH URL with default port", "ssh://github.com/shipwright-io/build.git", "github.com", 22, false),
		Entry("Check heritage SSH URL with custom port", "ssh://github.com:12134/shipwright-io/build.git", "github.com", 12134, false),
		Entry("Check SSH URL with default port", "git@github.com:shipwright-io/build.git", "github.com", 22, false),
		Entry("Check HTTP URL with default port", "http://github.com/shipwright-io/build.git", "github.com", 80, false),
		Entry("Check HTTPS URL with default port", "https://github.com/shipwright-io/build.git", "github.com", 443, false),
		Entry("Check HTTPS URL with custom port", "https://github.com:9443/shipwright-io/build.git", "github.com", 9443, false),
		Entry("Check HTTPS URL with credentials", "https://somebody:password@github.com/shipwright-io/build.git", "github.com", 443, false),
		Entry("Check invalid URL", "ftp://github.com/shipwright-io/build", "", 0, true),
	)

	DescribeTable("the source url validation errors",
		func(url string, expected types.GomegaMatcher) {
			Expect(git.ValidateGitURLExists(context.TODO(), url)).To(expected)
		},
		Entry("Check remote https public repository", "https://github.com/shipwright-io/build", BeNil()),
		Entry("Check remote fake https public repository", "https://github.com/shipwright-io/build-fake", Equal(errors.New("remote repository unreachable"))),
		Entry("Check invalid repository", "foobar", Equal(errors.New("invalid source url"))),
		Entry("Check git repository which requires authentication", "git@github.com:shipwright-io/build-fake.git", Equal(errors.New("the source url requires authentication"))),
		Entry("Check ssh repository which requires authentication", "ssh://github.com/shipwright-io/build-fake", Equal(errors.New("the source url requires authentication"))),
	)
})

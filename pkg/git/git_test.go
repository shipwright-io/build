// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"context"
	"errors"

	"github.com/go-git/go-git/v5/plumbing/transport"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Git", func() {
	originalListRemoteRefs := listRemoteRefs

	AfterEach(func() {
		listRemoteRefs = originalListRemoteRefs
	})

	DescribeTable("the extraction of hostname and port",
		func(url string, expectedHost string, expectedPort int, expectError bool) {
			host, port, err := ExtractHostnamePort(url)
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
		func(url string, remoteListErr error, expected types.GomegaMatcher, expectRemoteList bool) {
			remoteListCalled := false
			listRemoteRefs = func(_ context.Context, remoteURL string) error {
				remoteListCalled = true
				Expect(remoteURL).To(Equal(url))
				return remoteListErr
			}

			Expect(ValidateGitURLExists(context.TODO(), url)).To(expected)
			Expect(remoteListCalled).To(Equal(expectRemoteList))
		},
		Entry("Check remote https public repository", "https://github.com/shipwright-io/build", nil, BeNil(), true),
		Entry("Check remote fake https public repository", "https://github.com/shipwright-io/build-fake", transport.ErrAuthenticationRequired, Equal(errors.New("remote repository unreachable")), true),
		Entry("Check invalid repository", "foobar", nil, Equal(errors.New("invalid source url")), false),
		Entry("Check git repository which requires authentication", "git@github.com:shipwright-io/build-fake.git", nil, Equal(errors.New("the source url requires authentication")), false),
		Entry("Check ssh repository which requires authentication", "ssh://github.com/shipwright-io/build-fake", nil, Equal(errors.New("the source url requires authentication")), false),
	)
})

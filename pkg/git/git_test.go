// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package git_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/shipwright-io/build/pkg/git"
)

var _ = Describe("Git", func() {

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

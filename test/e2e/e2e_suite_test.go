// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/shipwright-io/build/test/utils"
)

var (
	testBuild *utils.TestBuild
	stop      = make(chan struct{})
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var (
	_ = BeforeSuite(func() {
		var (
			ok  bool
			err error
		)

		testBuild, err = utils.NewTestBuild()
		Expect(err).ToNot(HaveOccurred())

		testBuild.Namespace, ok = os.LookupEnv("TEST_NAMESPACE")
		Expect(ok).To(BeTrue(), "TEST_NAMESPACE should be set")
		Expect(testBuild.Namespace).ToNot(BeEmpty())

		// create the pipeline service account
		Logf("Creating the pipeline service account")
		createPipelineServiceAccount(testBuild)

		// create the container registry secret
		Logf("Creating the container registry secret")
		createContainerRegistrySecret(testBuild)
	})

	_ = AfterSuite(func() {
		close(stop)
	})
)

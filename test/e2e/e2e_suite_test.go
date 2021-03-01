// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/shipwright-io/build/test/integration/utils"
)

var (
	testBuild *utils.TestBuild
	stop      = make(chan struct{})

	clusterBuildStrategies = []string{
		"samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml",
		"samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_cr.yaml",
		"samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml",
		"samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml",
		"samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml",
		"samples/buildstrategy/source-to-image/buildstrategy_source-to-image-redhat_cr.yaml",
	}
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	var (
		ok  bool
		err error
	)

	testBuild, err = utils.NewTestBuild()
	Expect(err).ToNot(HaveOccurred())

	testBuild.Namespace, ok = os.LookupEnv("TEST_NAMESPACE")
	Expect(ok).To(BeTrue())

	operator := os.Getenv(EnvVarController)
	switch operator {
	case "start_local":
		Logf("Starting local operator")
		startLocalOperator(testBuild, stop)

	case "managed_outside":
		Logf("Using operator that is started outside of the e2e test suite.")

	default:
		Fail("Unexpected value for " + EnvVarController + ": '" + operator + "'")
	}

	// create the pipeline service account
	Logf("Creating the pipeline service account")
	createPipelineServiceAccount(testBuild)

	// create the container registry secret
	Logf("Creating the container registry secret")
	createContainerRegistrySecret(testBuild)

	if os.Getenv(EnvVarCreateGlobalObjects) == "true" {
		// create cluster build strategies
		Logf("Creating cluster build strategies")
		for _, clusterBuildStrategy := range clusterBuildStrategies {
			Logf("Creating cluster build strategy %s", clusterBuildStrategy)
			cbs, err := clusterBuildStrategyTestData(clusterBuildStrategy)
			Expect(err).ToNot(HaveOccurred(), "Error retrieving cluster buildstrategy test data")

			createClusterBuildStrategy(testBuild, cbs)
		}
	} else {
		Logf("Build strategy creation skipped.")
	}

	return nil

}, func(_ []byte) {
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	close(stop)
})

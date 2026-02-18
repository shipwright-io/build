// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

var _ = Describe("For a Kubernetes cluster with Tekton and build installed", func() {
	var (
		testID string
		err    error

		build    *buildv1beta1.Build
		buildRun *buildv1beta1.BuildRun
	)

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			printTestFailureDebugInfo(testBuild, testBuild.Namespace, testID)

		} else if buildRun != nil {
			validateServiceAccountDeletion(buildRun, testBuild.Namespace)
		}

		if buildRun != nil {
			Expect(testBuild.DeleteBR(buildRun.Name)).To(Succeed())
			buildRun = nil
		}

		if build != nil {
			Expect(testBuild.DeleteBuild(build.Name)).To(Succeed())
			build = nil
		}
	})

	Context("when a Buildah build with git depth is defined", func() {
		BeforeEach(func() {
			testID = generateTestID("git-depth")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_buildah_cr_custom_context+dockerfile.yaml",
			)
		})

		It("successfully runs a build with limited git history", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_buildah_cr_custom_context+dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")
			appendRegistryInsecureParamValue(build, buildRun)

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})
})

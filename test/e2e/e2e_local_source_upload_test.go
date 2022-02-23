// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/test/utils"
)

var _ = Describe("For a Kubernetes cluster with Tekton and build installed", func() {
	var (
		testID   string
		build    *buildv1alpha1.Build
		buildRun *buildv1alpha1.BuildRun
	)

	AfterEach(func() {
		if buildRun != nil {
			testBuild.DeleteBR(buildRun.Name)
			buildRun = nil
		}

		if build != nil {
			testBuild.DeleteBuild(build.Name)
			build = nil
		}
	})

	Context("when LocalCopy BuildSource is defined", func() {
		BeforeEach(func() {
			testID = generateTestID("local-copy")
			build = createBuild(
				testBuild,
				testID,
				"test/data/build_buildah_cr_local_source_upload.yaml",
			)
		})

		It("should generate LocalCopy TaskRun, using the waiter", func() {
			var err error
			buildRun, err = buildRunTestData(
				testBuild.Namespace,
				testID,
				"test/data/buildrun_buildah_cr_local_source_upload.yaml",
			)
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			validateWaiterBuildRun(testBuild, buildRun)
		})
	})
})

func getBuildRunStatusCondition(name types.NamespacedName) *buildv1alpha1.Condition {
	testBuildRun, err := testBuild.LookupBuildRun(name)
	Expect(err).ToNot(HaveOccurred(), "Error retrieving the BuildRun")

	if len(testBuildRun.Status.Conditions) == 0 {
		return nil
	}
	return testBuildRun.Status.GetCondition(buildv1alpha1.Succeeded)
}

// validateWaiterBuildRun assert the BuildRun informed will fail, since Waiter's timeout is reached
// and it causes the actual build process to fail as well.
func validateWaiterBuildRun(testBuild *utils.TestBuild, testBuildRun *buildv1alpha1.BuildRun) {
	err := testBuild.CreateBR(testBuildRun)
	Expect(err).ToNot(HaveOccurred(), "Failed to create BuildRun")

	buildRunName := types.NamespacedName{
		Namespace: testBuild.Namespace,
		Name:      testBuildRun.Name,
	}

	// making sure the taskrun is schedule and becomes a pod, since the build controller will transit
	// the object status from empty to unknown, when the actual build starts being executed
	Eventually(func() bool {
		condition := getBuildRunStatusCondition(buildRunName)
		if condition == nil {
			return false
		}
		Logf("BuildRun %q status %q...", buildRunName, condition.Status)
		return condition.Reason == "Running"
	}, time.Duration(1100*getTimeoutMultiplier())*time.Second, 5*time.Second).
		Should(BeTrue(), "BuildRun should start running")

	// asserting the waiter step will end up in timeout, in other words, the build is terminated with
	// the reason "failed"
	Eventually(func() string {
		condition := getBuildRunStatusCondition(buildRunName)
		Expect(condition).ToNot(BeNil())
		Logf("BuildRun %q condition %v...", buildRunName, condition)
		return condition.Reason
	}, time.Duration(90*time.Second), 10*time.Second).
		Should(Equal("Failed"), "BuildRun should end up in timeout")
}

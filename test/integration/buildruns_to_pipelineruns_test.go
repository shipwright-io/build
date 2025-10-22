// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	test "github.com/shipwright-io/build/test/v1beta1_samples"
)

var _ = Describe("Integration tests BuildRuns and PipelineRuns", func() {
	var (
		cbsObject      *v1beta1.ClusterBuildStrategy
		buildObject    *v1beta1.Build
		buildRunObject *v1beta1.BuildRun
	)
	// Delete the ClusterBuildStrategies after each test case
	AfterEach(func() {

		_, err = tb.GetBuild(buildObject.Name)
		if err == nil {
			Expect(tb.DeleteBuild(buildObject.Name)).To(BeNil())
		}

		err := tb.DeleteClusterBuildStrategy(cbsObject.Name)
		Expect(err).To(BeNil())

		_, err = tb.GetBR(buildRunObject.Name)
		if err == nil {
			Expect(tb.DeleteBR(buildRunObject.Name)).To(BeNil())
		}
	})
	Context("when a buildrun is created", func() {
		It("should create a pipelinerun that is owned by the buildrun", func() {
			cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategyNoOp))
			Expect(err).To(BeNil())

			err = tb.CreateClusterBuildStrategy(cbsObject)
			Expect(err).To(BeNil())

			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, STRATEGY+tb.Namespace, []byte(test.BuildCBSMinimal))
			Expect(err).To(BeNil())
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, BUILD+tb.Namespace, []byte(test.MinimalBuildRun))
			Expect(err).To(BeNil())
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			_, err = tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			// Wait for the BuildRun to have an executor reference
			Eventually(func() error {
				br, err := tb.GetBR(buildRunObject.Name)
				if err != nil {
					return err
				}
				if br.Status.Executor == nil || br.Status.Executor.Name == "" {
					return fmt.Errorf("BuildRun executor not set yet")
				}
				return nil
			}, "30s", "1s").Should(Succeed())

			pipelinerunObject, err := tb.GetPipelineRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			// Check that the taskrun is owned by the buildrun
			Expect(pipelinerunObject.OwnerReferences).To(HaveLen(1), "taskrun should have exactly one owner reference")
			ownerRef := pipelinerunObject.OwnerReferences[0]
			Expect(ownerRef.Kind).To(Equal("BuildRun"), "taskrun should have a buildrun owner reference")
			Expect(ownerRef.Name).To(Equal(buildRunObject.Name), "taskrun should have a buildrun owner reference")

		})
	})
	Context("when condition status true", func() {
		It("should reflect succeeded reason in the buildrun condition", func() {
			cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategyNoOp))
			Expect(err).To(BeNil())

			err = tb.CreateClusterBuildStrategy(cbsObject)
			Expect(err).To(BeNil())

			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, STRATEGY+tb.Namespace, []byte(test.BuildCBSMinimal))
			Expect(err).To(BeNil())
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, BUILD+tb.Namespace, []byte(test.MinimalBuildRun))
			Expect(err).To(BeNil())
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			// Wait for the BuildRun to complete and verify it succeeded
			buildRun, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(buildRun.Status.CompletionTime).ToNot(BeNil())

			// Verify the BuildRun used a PipelineRun executor
			Expect(buildRun.Status.Executor).ToNot(BeNil())
			Expect(buildRun.Status.Executor.Kind).To(Equal("PipelineRun"))
			Expect(buildRun.Status.Executor.Name).ToNot(BeEmpty())

			reason, err := tb.GetBRReason(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(reason).To(Equal("Succeeded"))
		})
	})

	Context("when condition status is false", func() {
		It("reflects a timeout", func() {
			cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategySingleStepKaniko))
			Expect(err).To(BeNil())

			err = tb.CreateClusterBuildStrategy(cbsObject)
			Expect(err).To(BeNil())

			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, STRATEGY+tb.Namespace, []byte(test.BuildCBSWithShortTimeOut))
			Expect(err).To(BeNil())
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, BUILD+tb.Namespace, []byte(test.MinimalBuildRun))
			Expect(err).To(BeNil())
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			buildRun, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).ToNot(HaveOccurred())

			condition := buildRun.Status.GetCondition(v1beta1.Succeeded)
			Expect(condition.Status).To(Equal(corev1.ConditionFalse))
			Expect(condition.Reason).To(Equal("BuildRunTimeout"))
			Expect(condition.Message).To(Equal(fmt.Sprintf("BuildRun %s failed to finish within %v", buildRun.Name, buildObject.Spec.Timeout.Duration)))
		})
	})

	Context("when pipelinerun status changes", func() {
		It("should synchronize running status to buildrun", func() {
			cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategyNoOp))
			Expect(err).To(BeNil())

			err = tb.CreateClusterBuildStrategy(cbsObject)
			Expect(err).To(BeNil())

			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, STRATEGY+tb.Namespace, []byte(test.BuildCBSMinimal))
			Expect(err).To(BeNil())
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, BUILD+tb.Namespace, []byte(test.MinimalBuildRun))
			Expect(err).To(BeNil())
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			// Wait for BuildRun to start
			buildRun, err := tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			// Verify BuildRun has start time set
			Expect(buildRun.Status.StartTime).ToNot(BeNil(), "BuildRun should have start time when PipelineRun starts")

			// Verify BuildRun condition reflects running state
			condition := buildRun.Status.GetCondition(v1beta1.Succeeded)
			Expect(condition).ToNot(BeNil(), "BuildRun should have Succeeded condition")
			Expect(condition.Status).To(Equal(corev1.ConditionUnknown), "BuildRun should be in Unknown state while running")

			// Get the PipelineRun and verify it's running
			pipelineRun, err := tb.GetPipelineRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(pipelineRun.Status.StartTime).ToNot(BeNil(), "PipelineRun should have start time")
		})
	})
})

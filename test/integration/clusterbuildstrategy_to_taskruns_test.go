// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	test "github.com/shipwright-io/build/test/v1beta1_samples"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

var _ = Describe("Integration tests ClusterBuildStrategies and TaskRuns", func() {
	var (
		cbsObject      *v1beta1.ClusterBuildStrategy
		buildObject    *v1beta1.Build
		buildRunObject *v1beta1.BuildRun
		buildSample    []byte
		buildRunSample []byte
	)

	// Load the BuildStrategies before each test case
	BeforeEach(func() {
		cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategyWithAnnotations))
		Expect(err).To(BeNil())

		err = tb.CreateClusterBuildStrategy(cbsObject)
		Expect(err).To(BeNil())
	})

	// Delete the BuildStrategies after each test case
	AfterEach(func() {
		_, err = tb.GetBuild(buildObject.Name)
		if err == nil {
			Expect(tb.DeleteBuild(buildObject.Name)).To(BeNil())
		}

		err := tb.DeleteClusterBuildStrategy(cbsObject.Name)
		Expect(err).To(BeNil())
	})

	// Override the Build and BuildRun CRD instances to use
	// before an It() statement is executed
	JustBeforeEach(func() {
		if buildSample != nil {
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, STRATEGY+tb.Namespace, buildSample)
			Expect(err).To(BeNil())
		}

		if buildRunSample != nil {
			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, BUILD+tb.Namespace, buildRunSample)
			Expect(err).To(BeNil())
		}
	})

	Context("when a buildrun is created", func() {
		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should create a taskrun with the correct annotations", func() {
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			taskRun, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			Expect(taskRun.Annotations["kubernetes.io/ingress-bandwidth"]).To(Equal("1M"))
			_, containsKey := taskRun.Annotations["clusterbuildstrategy.shipwright.io/dummy"]
			Expect(containsKey).To(BeFalse())
			_, containsKey = taskRun.Annotations["kubectl.kubernetes.io/last-applied-configuration"]
			Expect(containsKey).To(BeFalse())
		})
	})

	Context("when a build with stepResources using ClusterBuildStrategy is created", func() {
		BeforeEach(func() {
			// Use a ClusterBuildStrategy with defined resources
			cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategyWithResources))
			Expect(err).To(BeNil())

			// Delete existing CBS and create the new one
			_ = tb.DeleteClusterBuildStrategy(cbsObject.Name)
			err = tb.CreateClusterBuildStrategy(cbsObject)
			Expect(err).To(BeNil())

			buildSample = []byte(test.BuildWithStepResourcesClusterBuildStrategy)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should create a taskrun with the correct stepResources overridden", func() {
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			taskRun, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			// Find the build-step and verify its resources were overridden
			var buildStep *pipelineapi.Step
			for i := range taskRun.Spec.TaskSpec.Steps {
				if taskRun.Spec.TaskSpec.Steps[i].Name == "build-step" {
					buildStep = &taskRun.Spec.TaskSpec.Steps[i]
					break
				}
			}

			Expect(buildStep).ToNot(BeNil())
			// Verify that the step resources were overridden
			Expect(buildStep.ComputeResources.Requests.Cpu().String()).To(Equal("500m"))
			Expect(buildStep.ComputeResources.Requests.Memory().String()).To(Equal("512Mi"))
			Expect(buildStep.ComputeResources.Limits.Cpu().String()).To(Equal("1"))
			Expect(buildStep.ComputeResources.Limits.Memory().String()).To(Equal("1Gi"))

			// Find the push-step and verify it retains strategy defaults
			var pushStep *pipelineapi.Step
			for i := range taskRun.Spec.TaskSpec.Steps {
				if taskRun.Spec.TaskSpec.Steps[i].Name == "push-step" {
					pushStep = &taskRun.Spec.TaskSpec.Steps[i]
					break
				}
			}

			Expect(pushStep).ToNot(BeNil())
			// Verify that the push step retains strategy defaults (requests and limits)
			Expect(pushStep.ComputeResources.Requests.Cpu().String()).To(Equal("50m"))
			Expect(pushStep.ComputeResources.Requests.Memory().String()).To(Equal("64Mi"))
			Expect(pushStep.ComputeResources.Limits.Cpu().String()).To(Equal("100m"))
			Expect(pushStep.ComputeResources.Limits.Memory().String()).To(Equal("128Mi"))
		})
	})
})

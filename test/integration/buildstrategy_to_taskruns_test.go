// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/test"
)

var _ = Describe("Integration tests BuildStrategies and TaskRuns", func() {
	var (
		bsObject       *v1alpha1.BuildStrategy
		buildObject    *v1alpha1.Build
		buildRunObject *v1alpha1.BuildRun
		buildSample    []byte
		buildRunSample []byte
	)

	// Load the BuildStrategies before each test case
	BeforeEach(func() {
		bsObject, err = tb.Catalog.LoadBuildStrategyFromBytes([]byte(test.BuildahBuildStrategySingleStep))
		Expect(err).To(BeNil())

		err = tb.CreateBuildStrategy(bsObject)
		Expect(err).To(BeNil())
	})

	// Delete the BuildStrategies after each test case
	AfterEach(func() {

		_, err = tb.GetBuild(buildObject.Name)
		if err == nil {
			Expect(tb.DeleteBuild(buildObject.Name)).To(BeNil())
		}

		err := tb.DeleteBuildStrategy(bsObject.Name)
		Expect(err).To(BeNil())
	})

	// Override the Build and BuildRun CRD instances to use
	// before an It() statement is executed
	JustBeforeEach(func() {
		if buildSample != nil {
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, bsObject.Name, buildSample)
			Expect(err).To(BeNil())
		}

		if buildRunSample != nil {
			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, BUILD+tb.Namespace, buildRunSample)
			Expect(err).To(BeNil())
		}
	})

	Context("when a buildrun is created", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildBSMinimal)
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

			Expect(taskRun.Annotations["kubernetes.io/egress-bandwidth"]).To(Equal("1M"))
			Expect(taskRun.Annotations["kubernetes.io/ingress-bandwidth"]).To(Equal("1M"))
			_, containsKey := taskRun.Annotations["clusterbuildstrategy.shipwright.io/dummy"]
			Expect(containsKey).To(BeFalse())
			_, containsKey = taskRun.Annotations["kubectl.kubernetes.io/last-applied-configuration"]
			Expect(containsKey).To(BeFalse())
		})
	})

	Context("buildstrategy with defined parameters", func() {

		BeforeEach(func() {
			// Create a Strategy with parameters
			bsObject, err = tb.Catalog.LoadBuildStrategyFromBytes(
				[]byte(test.BuildStrategyWithParameters),
			)
			Expect(err).To(BeNil())

			err = tb.CreateBuildStrategy(bsObject)
			Expect(err).To(BeNil())

			// Create a minimal BuildRun
			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRun),
			)
			Expect(err).To(BeNil())
		})

		var constructedParam = func(paramName string, val string) v1beta1.Param {
			return v1beta1.Param{
				Name: paramName,
				Value: v1beta1.ArrayOrString{
					Type:      v1beta1.ParamTypeString,
					StringVal: val,
				},
			}
		}

		var constructBuildObjectAndWait = func(b *v1alpha1.Build) {
			// Create the Build object in-cluster
			Expect(tb.CreateBuild(b)).To(BeNil())

			// Wait until the Build object is validated
			_, err = tb.GetBuildTillValidation(b.Name)
			Expect(err).To(BeNil())
		}

		var constructBuildRunObjectAndWait = func(br *v1alpha1.BuildRun) {
			// Create the BuildRun object in-cluster
			Expect(tb.CreateBR(br)).To(BeNil())

			// Wait until the BuildRun is registered
			_, err = tb.GetBRTillStartTime(br.Name)
			Expect(err).To(BeNil())

		}

		It("uses sleep-time param if specified in the Build with buildstrategy", func() {
			// Set BuildWithSleepTimeParam with a value of 30
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObject.Name,
				[]byte(test.BuildWithSleepTimeParam),
			)
			Expect(err).To(BeNil())

			constructBuildObjectAndWait(buildObject)

			constructBuildRunObjectAndWait(buildRunObject)

			taskRun, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			Expect(taskRun.Spec.Params).To(ContainElement(constructedParam("sleep-time", "30")))

		})

		It("overrides sleep-time param if specified in the BuildRun", func() {
			// Set BuildWithSleepTimeParam with a value of 30
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObject.Name,
				[]byte(test.BuildWithSleepTimeParam),
			)
			Expect(err).To(BeNil())

			constructBuildObjectAndWait(buildObject)

			// Construct a BuildRun object that references the previous Build
			// without parameters definitions
			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRunWithParams),
			)
			Expect(err).To(BeNil())

			constructBuildRunObjectAndWait(buildRunObject)

			taskRun, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			Expect(taskRun.Spec.Params).To(ContainElement(constructedParam("sleep-time", "15")))

		})
		It("fails the TaskRun generation if the buildRun specifies a reserved system parameter", func() {
			// Build without params
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObject.Name,
				[]byte(test.BuildBSMinimal),
			)
			Expect(err).To(BeNil())

			constructBuildObjectAndWait(buildObject)

			// Construct a BuildRun object that references the previous Build
			// without usage of reserved params
			buildRunObjectWithReservedParams, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRunWithReservedParams),
			)
			Expect(err).To(BeNil())

			// Create the BuildRun object in-cluster
			Expect(tb.CreateBR(buildRunObjectWithReservedParams)).To(BeNil())

			// Wait until the BuildRun is registered
			br, err := tb.GetBRTillCompletion(buildRunObjectWithReservedParams.Name)
			Expect(err).To(BeNil())

			Expect(br.Status.GetCondition(v1alpha1.Succeeded).GetReason()).To(Equal("TaskRunGenerationFailed"))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).GetMessage()).To(ContainSubstring("restricted parameters in use"))
		})

		It("add params from buildRun if they are not defined in the Build", func() {
			// Build without params
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObject.Name,
				[]byte(test.BuildBSMinimal),
			)
			Expect(err).To(BeNil())

			constructBuildObjectAndWait(buildObject)

			// Construct a BuildRun object that references the previous Build
			// without parameters definitions
			buildRunObject, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRunWithParams),
			)
			Expect(err).To(BeNil())

			constructBuildRunObjectAndWait(buildRunObject)

			_, err = tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

		})

		It("fails the Build due to the usage of a restricted parameter name", func() {
			// Build using shipwright restricted params
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObject.Name,
				[]byte(test.BuildWithRestrictedParam),
			)
			Expect(err).To(BeNil())

			// Create the Build object in-cluster
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// Wait until the Build object is validated
			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(buildObject.Status.Reason).To(Equal(v1alpha1.RestrictedParametersInUse))
			Expect(buildObject.Status.Message).To(ContainSubstring("restricted parameters in use"))
		})

		It("fails the Build due to the definition of an undefined param in the strategy", func() {
			// Build using undefined parameter in the referenced strategy
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObject.Name,
				[]byte(test.BuildWithUndefinedParam),
			)
			Expect(err).To(BeNil())

			// Create the Build object in-cluster
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// Wait until the Build object is validated
			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(buildObject.Status.Reason).To(Equal(v1alpha1.UndefinedParameter))
			Expect(buildObject.Status.Message).To(ContainSubstring("parameter not defined in the strategies"))
		})

		It("allows a user to set an empty string on parameter without default", func() {

			// Create a BuildStrategy with a parameter without default value
			bsObjectOverride, err := tb.Catalog.LoadBuildStrategyFromBytes(
				[]byte(test.BuildStrategyWithoutDefaultInParameter),
			)
			Expect(err).To(BeNil())

			err = tb.CreateBuildStrategy(bsObjectOverride)
			Expect(err).To(BeNil())

			// Build that uses an empty string on the single param
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObjectOverride.Name,
				[]byte(test.BuildWithEmptyStringParam),
			)
			Expect(err).To(BeNil())

			// Create the Build object in-cluster
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// Wait until the Build object is validated
			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Construct a BuildRun object that references the previous Build
			// without parameters definitions
			buildRunObject, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRun),
			)
			Expect(err).To(BeNil())

			constructBuildRunObjectAndWait(buildRunObject)

			_, err = tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			taskRun, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			// Validate that the TaskSpec parameter have no default value
			Expect(taskRun.Spec.TaskSpec.Params).To(ContainElement(v1beta1.ParamSpec{
				Name:        "sleep-time",
				Type:        v1beta1.ParamTypeString,
				Description: "time in seconds for sleeping",
				Default:     nil,
			}))

			// Validate that the TaskRun param have an empty string as the value
			Expect(taskRun.Spec.Params).To(ContainElement(v1beta1.Param{
				Name: "sleep-time",
				Value: v1beta1.ArrayOrString{
					Type:      v1beta1.ParamTypeString,
					StringVal: "",
				},
			}))
		})

		It("fails the taskrun when a strategy parameter value is never specified", func() {

			// Create a BuildStrategy with a parameter without default value
			bsObjectOverride, err := tb.Catalog.LoadBuildStrategyFromBytes(
				[]byte(test.BuildStrategyWithoutDefaultInParameter),
			)
			Expect(err).To(BeNil())

			err = tb.CreateBuildStrategy(bsObjectOverride)
			Expect(err).To(BeNil())

			// Build that does not define a param
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObjectOverride.Name,
				[]byte(test.BuildBSMinimal),
			)
			Expect(err).To(BeNil())

			// Create the Build object in-cluster
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// Wait until the Build object is validated
			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Construct a BuildRun object that references the previous Build
			// without parameters definitions
			buildRunObject, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRun),
			)
			Expect(err).To(BeNil())

			// Create the BuildRun object in-cluster
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())

			Expect(br.Status.GetCondition(v1alpha1.Succeeded).GetReason()).To(Equal("TaskRunGenerationFailed"))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).GetMessage()).To(Equal("parameters without a value definition: sleep-time"))

		})

	})
})

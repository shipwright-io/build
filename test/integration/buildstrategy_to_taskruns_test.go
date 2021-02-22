// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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
			_, containsKey := taskRun.Annotations["clusterbuildstrategy.build.dev/dummy"]
			Expect(containsKey).To(BeFalse())
			_, containsKey = taskRun.Annotations["kubectl.kubernetes.io/last-applied-configuration"]
			Expect(containsKey).To(BeFalse())
		})
	})
})

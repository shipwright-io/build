// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/test"
	"knative.dev/pkg/apis"
)

var _ = Describe("Integration tests BuildRuns and TaskRuns", func() {
	var (
		cbsObject      *v1alpha1.ClusterBuildStrategy
		buildObject    *v1alpha1.Build
		buildRunObject *v1alpha1.BuildRun
		buildSample    []byte
		buildRunSample []byte
		ctx            context.Context
	)

	// Load the ClusterBuildStrategies before each test case
	BeforeEach(func() {
		ctx = context.Background()
		cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategySingleStep))
		Expect(err).To(BeNil())

		err = tb.CreateClusterBuildStrategy(ctx, cbsObject)
		Expect(err).To(BeNil())
	})

	// Delete the ClusterBuildStrategies after each test case
	AfterEach(func() {

		_, err = tb.GetBuild(ctx, buildObject.Name)
		if err == nil {
			Expect(tb.DeleteBuild(ctx, buildObject.Name)).To(BeNil())
		}

		err := tb.DeleteClusterBuildStrategy(ctx, cbsObject.Name)
		Expect(err).To(BeNil())
	})

	// Override the Builds and BuildRuns CRDs instances to use
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

		It("should reflect a Pending and Running reason", func() {

			Expect(tb.CreateBuild(ctx, buildObject)).To(BeNil())

			Expect(tb.CreateBR(ctx, buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillStartTime(ctx, buildRunObject.Name)
			Expect(err).To(BeNil())

			err = tb.GetTRTillDesiredReason(buildRunObject.Name, "Pending")
			Expect(err).To(BeNil())

			err = tb.GetBRTillDesiredReason(ctx, buildRunObject.Name, "Pending")
			Expect(err).To(BeNil())

			err = tb.GetTRTillDesiredReason(buildRunObject.Name, "Running")
			Expect(err).To(BeNil())

			err = tb.GetBRTillDesiredReason(ctx, buildRunObject.Name, "Running")
			Expect(err).To(BeNil())

		})
	})

	Context("when a buildrun is created but fails because of a timeout", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSWithShortTimeOut)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should reflect a TaskRunTimeout reason and Completion time on timeout", func() {

			Expect(tb.CreateBuild(ctx, buildObject)).To(BeNil())

			Expect(tb.CreateBR(ctx, buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillCompletion(ctx, buildRunObject.Name)
			Expect(err).To(BeNil())

			err = tb.GetTRTillDesiredReason(buildRunObject.Name, "TaskRunTimeout")
			Expect(err).To(BeNil())

			tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			err = tb.GetBRTillDesiredReason(ctx, buildRunObject.Name, fmt.Sprintf("TaskRun \"%s\" failed to finish within \"5s\"", tr.Name))
			Expect(err).To(BeNil())

			tr, err = tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(tr.Status.CompletionTime).ToNot(BeNil())

		})
	})

	Context("when a buildrun is created with a wrong url", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSWithWrongURL)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should reflect a Failed reason and Completion on failure", func() {

			Expect(tb.CreateBuild(ctx, buildObject)).To(BeNil())

			Expect(tb.CreateBR(ctx, buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillCompletion(ctx, buildRunObject.Name)
			Expect(err).To(BeNil())

			err = tb.GetTRTillDesiredReason(buildRunObject.Name, "Failed")
			Expect(err).To(BeNil())

			tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			err = tb.GetBRTillDesiredReason(ctx, buildRunObject.Name, tr.Status.GetCondition(apis.ConditionSucceeded).Message)
			Expect(err).To(BeNil())

			tr, err = tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(tr.Status.CompletionTime).ToNot(BeNil())

		})
	})

	Context("when a buildrun is created and cancelled", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should reflect a TaskRunCancelled reason and no completionTime", func() {

			Expect(tb.CreateBuild(ctx, buildObject)).To(BeNil())

			Expect(tb.CreateBR(ctx, buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillStartTime(ctx, buildRunObject.Name)
			Expect(err).To(BeNil())

			tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			tr.Spec.Status = "TaskRunCancelled"

			tr, err = tb.UpdateTaskRun(tr)
			Expect(err).To(BeNil())

			err = tb.GetBRTillDesiredReason(ctx, buildRunObject.Name, fmt.Sprintf("TaskRun \"%s\" was cancelled", tr.Name))

			err = tb.GetTRTillDesiredReason(buildRunObject.Name, "TaskRunCancelled")
			Expect(err).To(BeNil())

		})
	})
})

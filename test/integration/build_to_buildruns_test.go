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
)

const (
	BUILD    = "build-"
	BUILDRUN = "buildrun-"
	STRATEGY = "strategy-"
)

var _ = Describe("Integration tests Build and BuildRuns", func() {

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

	Context("when a build with a short timeout is defined", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSWithShortTimeOut)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should fail the builRun with a Reason", func() {

			Expect(tb.CreateBuild(ctx, buildObject)).To(BeNil())

			Expect(tb.CreateBR(ctx, buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(ctx, buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.Reason).To(ContainSubstring("failed to finish within"))

		})
	})

	Context("when a buildrun defines build spec properties", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSWithShortTimeOut)
			buildRunSample = []byte(test.MinimalBuildRunWithTimeOut)
		})

		It("should be able to override the build timeout", func() {

			Expect(tb.CreateBuild(ctx, buildObject)).To(BeNil())

			Expect(tb.CreateBR(ctx, buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(ctx, buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.Reason).To(ContainSubstring("failed to finish within \"1s\""))
		})

		It("should be able to override the build output", func() {

			Expect(tb.CreateBuild(ctx, buildObject)).To(BeNil())

			buildRun, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRunWithOutput),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(ctx, buildRun)).To(BeNil())

			_, err = tb.GetBRTillStartTime(ctx, buildRun.Name)
			Expect(err).To(BeNil())

			tr, err := tb.GetTaskRunFromBuildRun(buildRun.Name)
			Expect(err).To(BeNil())

			Expect(tr.Spec.Resources.Outputs[0].PipelineResourceBinding.ResourceSpec.Params[0].Value).To(Equal("foobar.registry.com"))

		})
	})

	Context("when a build is deleted after the buildrun creation", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSWithBuildRunDeletion)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should delete the builRun automatically if builds uses the deletion annotation", func() {

			Expect(tb.CreateBuild(ctx, buildObject)).To(BeNil())

			Expect(tb.CreateBR(ctx, buildRunObject)).To(BeNil())

			// Wait for BR to get an Starttime
			_, err = tb.GetBRTillStartTime(ctx, buildRunObject.Name)
			Expect(err).To(BeNil())

			//Delete Build
			Expect(tb.DeleteBuild(ctx, buildObject.Name)).To(BeNil())

			// Wait for deletion of BuildRun
			brDel, err := tb.GetBRTillDeletion(ctx, buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(brDel).To(Equal(true))

		})

		// TODO: not sure if this is a bug or we added this behaviour at some point, smells fishy
		It("does not fail the buildrun and nothing is reflected in the buildrun status", func() {
			build, err := tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildCBSMinimal),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBuild(ctx, build)).To(BeNil())

			Expect(tb.CreateBR(ctx, buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillStartTime(ctx, buildRunObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.DeleteBuild(ctx, BUILD+tb.Namespace)).To(BeNil())

			br, err = tb.GetBR(ctx, buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.CompletionTime).To(BeNil())

		})
	})

	Context("when a build is deleted before the buildrun creation", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("fails the buildrun with a reason and no startime", func() {

			Expect(tb.CreateBuild(ctx, buildObject)).To(BeNil())

			err = tb.DeleteBuild(ctx, BUILD+tb.Namespace)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(ctx, buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(ctx, buildRunObject.Name)
			Expect(err).To(BeNil())

			br, err = tb.GetBR(ctx, buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.Reason).To(Equal(fmt.Sprintf("Build.build.dev \"%s\" not found", BUILD+tb.Namespace)))
			Expect(br.Status.StartTime).To(BeNil())

		})
	})

	Context("when a build is not registered correctly", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimalWithFakeSecret)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("fails the buildrun with a proper error in Reason", func() {

			Expect(tb.CreateBuild(ctx, buildObject)).To(BeNil())

			Expect(tb.CreateBR(ctx, buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(ctx, buildRunObject.Name)
			Expect(err).To(BeNil())

			Expect(br.Status.Reason).To(Equal(fmt.Sprintf("The Build is not registered correctly, build: %s, registered status: False, reason: secret fake-secret does not exist", BUILD+tb.Namespace)))
		})
	})

	Context("when a buildrun reference an unknown build", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
		})

		It("fails the buildrun with a not found Reason", func() {

			Expect(tb.CreateBuild(ctx, buildObject)).To(BeNil())

			buildRun, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace+"foobar",
				[]byte(test.MinimalBuildRun),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(ctx, buildRun)).To(BeNil())

			br, err := tb.GetBRTillCompletion(ctx, buildRun.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.CompletionTime).ToNot(BeNil())
			Expect(br.Status.StartTime).To(BeNil())
			Expect(br.Status.Reason).To(Equal(fmt.Sprintf("Build.build.dev \"%s\" not found", BUILD+tb.Namespace+"foobar")))
		})
	})

	Context("when multiple buildruns reference a build", func() {
		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
		})

		It("creates one tr per buildrun with the original build data", func() {

			Expect(tb.CreateBuild(ctx, buildObject)).To(BeNil())

			buildRun01, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace+"01",
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRun),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(ctx, buildRun01)).To(BeNil())

			buildRun02, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace+"02",
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRun),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(ctx, buildRun02)).To(BeNil())

			_, err = tb.GetBRTillStartTime(ctx, buildRun01.Name)
			Expect(err).To(BeNil())

			_, err = tb.GetBRTillStartTime(ctx, buildRun02.Name)
			Expect(err).To(BeNil())

			tr01, err := tb.GetTaskRunFromBuildRun(buildRun01.Name)
			Expect(err).To(BeNil())
			Expect(tr01.Spec.Resources.Inputs[0].PipelineResourceBinding.ResourceSpec.Params[0].Value).To(Equal("https://github.com/sbose78/taxi"))

			tr02, err := tb.GetTaskRunFromBuildRun(buildRun02.Name)
			Expect(err).To(BeNil())
			Expect(tr02.Spec.Resources.Inputs[0].PipelineResourceBinding.ResourceSpec.Params[0].Value).To(Equal("https://github.com/sbose78/taxi"))

		})
	})
})

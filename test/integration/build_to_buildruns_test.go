// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

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
	)

	// Load the ClusterBuildStrategies before each test case
	BeforeEach(func() {
		cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategySingleStep))
		Expect(err).To(BeNil())

		err = tb.CreateClusterBuildStrategy(cbsObject)
		Expect(err).To(BeNil())

	})
	// Delete the ClusterBuildStrategies after each test case
	AfterEach(func() {

		_, err = tb.GetBuild(buildObject.Name)
		if err == nil {
			Expect(tb.DeleteBuild(buildObject.Name)).To(BeNil())
		}

		err := tb.DeleteClusterBuildStrategy(cbsObject.Name)
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

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.Reason).To(ContainSubstring("failed to finish within"))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionFalse))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Reason).To(Equal("BuildRunTimeout"))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Message).To(ContainSubstring("failed to finish within"))
		})
	})

	Context("when a buildrun defines build spec properties", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSWithShortTimeOut)
			buildRunSample = []byte(test.MinimalBuildRunWithTimeOut)
		})

		It("should be able to override the build timeout", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.Reason).To(ContainSubstring("failed to finish within \"1s\""))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionFalse))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Reason).To(Equal("BuildRunTimeout"))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Message).To(ContainSubstring("failed to finish within"))
		})

		It("should be able to override the build output", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			buildRun, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRunWithOutput),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRun)).To(BeNil())

			_, err = tb.GetBRTillStartTime(buildRun.Name)
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

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			// Wait for BR to get an Starttime
			_, err = tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			//Delete Build
			Expect(tb.DeleteBuild(buildObject.Name)).To(BeNil())

			// Wait for deletion of BuildRun
			brDel, err := tb.GetBRTillDeletion(buildRunObject.Name)
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

			Expect(tb.CreateBuild(build)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.DeleteBuild(BUILD + tb.Namespace)).To(BeNil())

			br, err = tb.GetBR(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.CompletionTime).To(BeNil())
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Type).To(Equal(v1alpha1.Succeeded))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionUnknown))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Reason).To(
				// BuildRun reason can be ExceededNodeResources
				// if the Tekton TaskRun Pod is queued due to
				// insufficient cluster resources.
				Or(Equal("Pending"), Equal("ExceededNodeResources")))
		})
	})

	Context("when a build is deleted before the buildrun creation", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("fails the buildrun with a reason and no startime", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			err = tb.DeleteBuild(BUILD + tb.Namespace)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())

			br, err = tb.GetBR(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.Reason).To(Equal(fmt.Sprintf("Build.build.dev \"%s\" not found", BUILD+tb.Namespace)))
			Expect(br.Status.StartTime).To(BeNil())
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionFalse))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Reason).To(Equal("Failed"))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Message).To(ContainSubstring("not found"))

		})
	})

	Context("when a build is not registered correctly", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimalWithFakeSecret)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("fails the buildrun with a proper error in Reason", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())

			Expect(br.Status.Reason).To(Equal(fmt.Sprintf("the Build is not registered correctly, build: %s, registered status: False, reason: SpecOutputSecretRefNotFound", BUILD+tb.Namespace)))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionFalse))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Reason).To(Equal("Failed"))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Message).To(ContainSubstring("Build is not registered correctly"))
		})
	})

	Context("when a buildrun reference an unknown build", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
		})

		It("fails the buildrun with a not found Reason", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			buildRun, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace+"foobar",
				[]byte(test.MinimalBuildRun),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRun)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRun.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.CompletionTime).ToNot(BeNil())
			Expect(br.Status.StartTime).To(BeNil())
			Expect(br.Status.Reason).To(Equal(fmt.Sprintf("Build.build.dev \"%s\" not found", BUILD+tb.Namespace+"foobar")))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionFalse))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Reason).To(Equal("Failed"))
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Message).To(ContainSubstring("not found"))
		})
	})

	Context("when multiple buildruns reference a build", func() {
		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
		})

		It("creates one tr per buildrun with the original build data", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			buildRun01, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace+"01",
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRun),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRun01)).To(BeNil())

			buildRun02, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace+"02",
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRun),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRun02)).To(BeNil())

			_, err = tb.GetBRTillStartTime(buildRun01.Name)
			Expect(err).To(BeNil())

			_, err = tb.GetBRTillStartTime(buildRun02.Name)
			Expect(err).To(BeNil())

			tr01, err := tb.GetTaskRunFromBuildRun(buildRun01.Name)
			Expect(err).To(BeNil())
			Expect(tr01.Spec.Resources.Inputs[0].PipelineResourceBinding.ResourceSpec.Params[0].Value).To(Equal("https://github.com/qu1queee/taxi"))

			tr02, err := tb.GetTaskRunFromBuildRun(buildRun02.Name)
			Expect(err).To(BeNil())
			Expect(tr02.Spec.Resources.Inputs[0].PipelineResourceBinding.ResourceSpec.Params[0].Value).To(Equal("https://github.com/qu1queee/taxi"))

		})
	})

	Context("when a build is annotated for deleting the buildrun", func() {
		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSWithBuildRunDeletion)
		})

		var ownerReferenceNames = func(list []metav1.OwnerReference) []string {
			var result = make([]string, len(list))
			for i, ownerReference := range list {
				result[i] = ownerReference.Name
			}
			return result
		}

		It("deletes the buildrun when the build is deleted", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			autoDeleteBuildRun, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRun),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(autoDeleteBuildRun)).To(BeNil())

			_, err = tb.GetBRTillStartTime(autoDeleteBuildRun.Name)
			Expect(err).To(BeNil())

			br, err := tb.GetBRTillOwner(BUILDRUN+tb.Namespace, buildObject.Name)
			Expect(err).To(BeNil())
			Expect(ownerReferenceNames(br.OwnerReferences)).Should(ContainElement(buildObject.Name))

			err = tb.DeleteBuild(BUILD + tb.Namespace)
			Expect(err).To(BeNil())

			buildIsDeleted, err := tb.GetBRTillDeletion(BUILDRUN + tb.Namespace)
			Expect(err).To(BeNil())
			Expect(buildIsDeleted).To(Equal(true))

		})

		It("does not deletes the buildrun if the annotation is changed", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			autoDeleteBuildRun, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRun),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(autoDeleteBuildRun)).To(BeNil())

			_, err = tb.GetBRTillStartTime(autoDeleteBuildRun.Name)
			Expect(err).To(BeNil())

			// we modify the annotation so automatic delete does not take place
			data := []byte(fmt.Sprintf(`{"metadata":{"annotations":{"%s":"false"}}}`, v1alpha1.AnnotationBuildRunDeletion))
			_, err = tb.PatchBuild(BUILD+tb.Namespace, data)
			Expect(err).To(BeNil())

			err = tb.DeleteBuild(BUILD + tb.Namespace)
			Expect(err).To(BeNil())

			br, err := tb.GetBRTillNotOwner(BUILDRUN+tb.Namespace, buildObject.Name)
			Expect(err).To(BeNil())
			Expect(ownerReferenceNames(br.OwnerReferences)).ShouldNot(ContainElement(buildObject.Name))

		})
		It("does not deletes the buildrun if the annotation is removed", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			autoDeleteBuildRun, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRun),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(autoDeleteBuildRun)).To(BeNil())

			_, err = tb.GetBRTillStartTime(autoDeleteBuildRun.Name)
			Expect(err).To(BeNil())

			// we remove the annotation so automatic delete does not take place, "/" is escaped by "~1" in a JSON pointer
			data := []byte(fmt.Sprintf(`[{"op":"remove","path":"/metadata/annotations/%s"}]`, strings.ReplaceAll(v1alpha1.AnnotationBuildRunDeletion, "/", "~1")))
			_, err = tb.PatchBuildWithPatchType(BUILD+tb.Namespace, data, types.JSONPatchType)
			Expect(err).To(BeNil())

			err = tb.DeleteBuild(BUILD + tb.Namespace)
			Expect(err).To(BeNil())

			br, err := tb.GetBRTillNotOwner(BUILDRUN+tb.Namespace, buildObject.Name)
			Expect(err).To(BeNil())
			Expect(ownerReferenceNames(br.OwnerReferences)).ShouldNot(ContainElement(buildObject.Name))

		})
		It("does delete the buildrun after several modifications of the annotation", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			autoDeleteBuildRun, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRun),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(autoDeleteBuildRun)).To(BeNil())

			// we modify the annotation for the automatic deletion to not take place
			data := []byte(fmt.Sprintf(`{"metadata":{"annotations":{"%s":"false"}}}`, v1alpha1.AnnotationBuildRunDeletion))
			_, err = tb.PatchBuild(BUILD+tb.Namespace, data)
			Expect(err).To(BeNil())

			patchedBuild, err := tb.GetBuild(BUILD + tb.Namespace)
			Expect(err).To(BeNil())
			Expect(patchedBuild.Annotations[v1alpha1.AnnotationBuildRunDeletion]).To(Equal("false"))

			_, err = tb.GetBRTillStartTime(autoDeleteBuildRun.Name)
			Expect(err).To(BeNil())

			// we modify the annotation one more time, to validate that the build should be deleted this time
			data = []byte(fmt.Sprintf(`{"metadata":{"annotations":{"%s":"true"}}}`, v1alpha1.AnnotationBuildRunDeletion))
			_, err = tb.PatchBuild(BUILD+tb.Namespace, data)
			Expect(err).To(BeNil())

			patchedBuild, err = tb.GetBuild(BUILD + tb.Namespace)
			Expect(err).To(BeNil())
			Expect(patchedBuild.Annotations[v1alpha1.AnnotationBuildRunDeletion]).To(Equal("true"))

			br, err := tb.GetBRTillOwner(BUILDRUN+tb.Namespace, buildObject.Name)
			Expect(err).To(BeNil())
			Expect(ownerReferenceNames(br.OwnerReferences)).Should(ContainElement(buildObject.Name))

			err = tb.DeleteBuild(BUILD + tb.Namespace)
			Expect(err).To(BeNil())

			buildIsDeleted, err := tb.GetBRTillDeletion(BUILDRUN + tb.Namespace)
			Expect(err).To(BeNil())
			Expect(buildIsDeleted).To(Equal(true))
		})
	})
})

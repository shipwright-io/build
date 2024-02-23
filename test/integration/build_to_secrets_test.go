// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	test "github.com/shipwright-io/build/test/v1beta1_samples"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Integration tests Build and referenced Secrets", func() {

	var (
		cbsObject   *v1beta1.ClusterBuildStrategy
		buildObject *v1beta1.Build
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
		err := tb.DeleteClusterBuildStrategy(cbsObject.Name)
		Expect(err).To(BeNil())
	})

	Context("when a build reference a secret with annotations for the spec output", func() {
		It("should validate the Build after secret deletion", func() {

			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildWithOutputRefSecret),
			)
			Expect(err).To(BeNil())

			sampleSecret := tb.Catalog.SecretWithAnnotation(*buildObject.Spec.Output.PushSecret, buildObject.Namespace)

			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillValidation(buildName)
			Expect(err).To(BeNil())
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SucceedStatus))

			// delete a secret
			Expect(tb.DeleteSecret(*buildObject.Spec.Output.PushSecret)).To(BeNil())

			// assert that the validation happened one more time
			buildObject, err = tb.GetBuildTillRegistration(buildName, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SpecOutputSecretRefNotFound))
			Expect(*buildObject.Status.Message).To(Equal(fmt.Sprintf("referenced secret %s not found", *buildObject.Spec.Output.PushSecret)))
		})

		It("should validate when a missing secret is recreated", func() {
			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildCBSMinimalWithFakeSecret),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillValidation(buildName)
			Expect(err).To(BeNil())
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SpecOutputSecretRefNotFound))
			Expect(*buildObject.Status.Message).To(Equal(fmt.Sprintf("referenced secret %s not found", *buildObject.Spec.Output.PushSecret)))

			sampleSecret := tb.Catalog.SecretWithAnnotation(*buildObject.Spec.Output.PushSecret, buildObject.Namespace)

			// generate resources
			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())

			// assert that the validation happened one more time
			buildObject, err = tb.GetBuildTillRegistration(buildName, corev1.ConditionTrue)
			Expect(err).To(BeNil())
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SucceedStatus))
		})
	})

	Context("when a build reference a secret without annotations for the spec output", func() {
		It("should not validate the Build after a secret deletion", func() {

			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildWithOutputRefSecret),
			)
			Expect(err).To(BeNil())

			sampleSecret := tb.Catalog.SecretWithoutAnnotation(*buildObject.Spec.Output.PushSecret, buildObject.Namespace)

			// generate resources
			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillValidation(buildName)
			Expect(err).To(BeNil())
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SucceedStatus))

			// delete a secret
			Expect(tb.DeleteSecret(*buildObject.Spec.Output.PushSecret)).To(BeNil())

			// assert that the validation happened one more time
			buildObject, err = tb.GetBuild(buildName)
			Expect(err).To(BeNil())
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
		})

		It("should not validate when a missing secret is recreated without annotation", func() {
			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildCBSMinimalWithFakeSecret),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillValidation(buildName)
			Expect(err).To(BeNil())
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SpecOutputSecretRefNotFound))
			Expect(*buildObject.Status.Message).To(Equal(fmt.Sprintf("referenced secret %s not found", *buildObject.Spec.Output.PushSecret)))

			sampleSecret := tb.Catalog.SecretWithoutAnnotation(*buildObject.Spec.Output.PushSecret, buildObject.Namespace)

			// generate resources
			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())

			// // assert that the validation happened one more time
			buildObject, err = tb.GetBuildTillRegistration(buildName, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SpecOutputSecretRefNotFound))
			Expect(*buildObject.Status.Message).To(Equal(fmt.Sprintf("referenced secret %s not found", *buildObject.Spec.Output.PushSecret)))
		})

		It("should validate when a missing secret is recreated with annotation", func() {
			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildCBSMinimalWithFakeSecret),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillValidation(buildName)
			Expect(err).To(BeNil())
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SpecOutputSecretRefNotFound))
			Expect(*buildObject.Status.Message).To(Equal(fmt.Sprintf("referenced secret %s not found", "fake-secret")))

			sampleSecret := tb.Catalog.SecretWithoutAnnotation(*buildObject.Spec.Output.PushSecret, buildObject.Namespace)

			// generate resources
			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())
			// validate build status again
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SpecOutputSecretRefNotFound))
			Expect(*buildObject.Status.Message).To(Equal(fmt.Sprintf("referenced secret %s not found", "fake-secret")))

			// we modify the annotation so automatic delete does not take place
			data := []byte(fmt.Sprintf(`{"metadata":{"annotations":{"%s":"true"}}}`, v1beta1.AnnotationBuildRefSecret))

			_, err = tb.PatchSecret(*buildObject.Spec.Output.PushSecret, data)
			Expect(err).To(BeNil())

			// // assert that the validation happened one more time
			buildObject, err = tb.GetBuildTillRegistration(buildName, corev1.ConditionTrue)
			Expect(err).To(BeNil())
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SucceedStatus))
			Expect(*buildObject.Status.Message).To(Equal("all validations succeeded"))
		})
	})

	Context("when a build reference a secret with annotations for the spec source", func() {
		It("should validate the Build after secret deletion", func() {

			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildWithSourceRefSecret),
			)
			Expect(err).To(BeNil())

			sampleSecret := tb.Catalog.SecretWithAnnotation(*buildObject.Spec.Source.Git.CloneSecret, buildObject.Namespace)

			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillValidation(buildName)
			Expect(err).To(BeNil())
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SucceedStatus))
			Expect(*buildObject.Status.Message).To(Equal("all validations succeeded"))

			// delete a secret
			Expect(tb.DeleteSecret(*buildObject.Spec.Source.Git.CloneSecret)).To(BeNil())

			// assert that the validation happened one more time
			buildObject, err = tb.GetBuildTillRegistration(buildName, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SpecSourceSecretRefNotFound))
			Expect(*buildObject.Status.Message).To(Equal(fmt.Sprintf("referenced secret %s not found", *buildObject.Spec.Source.Git.CloneSecret)))
		})

		It("should validate when a missing secret is recreated", func() {
			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildWithSourceRefSecret),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillValidation(buildName)
			Expect(err).To(BeNil())
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			// Status reason sometimes returns message "there are no secrets in namespace..."
			// Expect(buildObject.Status.Reason).To(Equal(fmt.Sprintf("secret %s does not exist", buildObject.Spec.Source.Credentials.Name)))

			sampleSecret := tb.Catalog.SecretWithAnnotation(*buildObject.Spec.Source.Git.CloneSecret, buildObject.Namespace)

			// generate resources
			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())

			// assert that the validation happened one more time
			buildObject, err = tb.GetBuildTillRegistration(buildName, corev1.ConditionTrue)
			Expect(err).To(BeNil())
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(*buildObject.Status.Reason).To(Equal(v1beta1.SucceedStatus))
		})
	})

	Context("when multiple builds reference a secret with annotations for the spec.source", func() {
		It("should validate the Builds after secret deletion", func() {

			// populate Build related vars
			firstBuildName := BUILD + tb.Namespace
			firstBuildObject, err := tb.Catalog.LoadBuildWithNameAndStrategy(
				firstBuildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildWithSourceRefSecret),
			)
			Expect(err).To(BeNil())

			// populate Build related vars
			secondBuildName := BUILD + tb.Namespace + "extra-build"
			secondBuildObject, err := tb.Catalog.LoadBuildWithNameAndStrategy(
				secondBuildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildWithSourceRefSecret),
			)
			Expect(err).To(BeNil())

			specSourceSecret := tb.Catalog.SecretWithAnnotation(*firstBuildObject.Spec.Source.Git.CloneSecret, firstBuildObject.Namespace)

			Expect(tb.CreateSecret(specSourceSecret)).To(BeNil())

			Expect(tb.CreateBuild(firstBuildObject)).To(BeNil())
			Expect(tb.CreateBuild(secondBuildObject)).To(BeNil())

			// wait until the Build finish the validation
			o, err := tb.GetBuildTillValidation(firstBuildName)
			Expect(err).To(BeNil())
			Expect(*o.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(*o.Status.Reason).To(Equal(v1beta1.SucceedStatus))

			o, err = tb.GetBuildTillValidation(secondBuildName)
			Expect(err).To(BeNil())
			Expect(*o.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(*o.Status.Reason).To(Equal(v1beta1.SucceedStatus))

			// delete a secret
			Expect(tb.DeleteSecret(specSourceSecret.Name)).To(BeNil())

			// assert that the validation happened one more time
			o, err = tb.GetBuildTillRegistration(firstBuildName, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			Expect(*o.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(*o.Status.Reason).To(Equal(v1beta1.SpecSourceSecretRefNotFound))
			Expect(*o.Status.Message).To(Equal(fmt.Sprintf("referenced secret %s not found", *firstBuildObject.Spec.Source.Git.CloneSecret)))

			// assert that the validation happened one more time
			o, err = tb.GetBuildTillRegistration(secondBuildName, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			Expect(*o.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(*o.Status.Reason).To(Equal(v1beta1.SpecSourceSecretRefNotFound))
			Expect(*o.Status.Message).To(Equal(fmt.Sprintf("referenced secret %s not found", *secondBuildObject.Spec.Source.Git.CloneSecret)))
		})
		It("should validate the Builds when a missing secret is recreated", func() {
			// populate Build related vars
			firstBuildName := BUILD + tb.Namespace
			firstBuildObject, err := tb.Catalog.LoadBuildWithNameAndStrategy(
				firstBuildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildWithSourceRefSecret),
			)
			Expect(err).To(BeNil())

			// populate Build related vars
			secondBuildName := BUILD + tb.Namespace + "extra-build"
			secondBuildObject, err := tb.Catalog.LoadBuildWithNameAndStrategy(
				secondBuildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildWithSourceRefSecret),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBuild(firstBuildObject)).To(BeNil())
			Expect(tb.CreateBuild(secondBuildObject)).To(BeNil())

			// wait until the Builds finish the validation
			buildObject, err := tb.GetBuildTillValidation(firstBuildName)
			Expect(err).To(BeNil())
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))

			buildObject, err = tb.GetBuildTillValidation(secondBuildName)
			Expect(err).To(BeNil())
			Expect(*buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))

			specSourceSecret := tb.Catalog.SecretWithAnnotation(*firstBuildObject.Spec.Source.Git.CloneSecret, firstBuildObject.Namespace)

			// generate resources
			Expect(tb.CreateSecret(specSourceSecret)).To(BeNil())

			// assert that the validation happened one more time
			o, err := tb.GetBuildTillRegistration(firstBuildName, corev1.ConditionTrue)
			Expect(err).To(BeNil())
			Expect(*o.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(*o.Status.Reason).To(Equal(v1beta1.SucceedStatus))

			o, err = tb.GetBuildTillRegistration(secondBuildName, corev1.ConditionTrue)
			Expect(err).To(BeNil())
			Expect(*o.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(*o.Status.Reason).To(Equal(v1beta1.SucceedStatus))
		})
	})
})

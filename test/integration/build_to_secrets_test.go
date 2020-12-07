// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/test"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Integration tests Build and referenced Secrets", func() {

	var (
		cbsObject   *v1alpha1.ClusterBuildStrategy
		buildObject *v1alpha1.Build
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

			sampleSecret := tb.Catalog.SecretWithAnnotation(buildObject.Spec.Output.SecretRef.Name, buildObject.Namespace)

			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillValidation(buildName)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(buildObject.Status.Reason).To(Equal("Succeeded"))

			// delete a secret
			Expect(tb.DeleteSecret(buildObject.Spec.Output.SecretRef.Name)).To(BeNil())

			// assert that the validation happened one more time
			buildObject, err = tb.GetBuildTillRegistration(buildName, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(buildObject.Status.Reason).To(Equal(fmt.Sprintf("secret %s does not exist", buildObject.Spec.Output.SecretRef.Name)))

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
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(buildObject.Status.Reason).To(Equal(fmt.Sprintf("secret %s does not exist", buildObject.Spec.Output.SecretRef.Name)))

			sampleSecret := tb.Catalog.SecretWithAnnotation(buildObject.Spec.Output.SecretRef.Name, buildObject.Namespace)

			// generate resources
			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())

			// assert that the validation happened one more time
			buildObject, err = tb.GetBuildTillRegistration(buildName, corev1.ConditionTrue)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(buildObject.Status.Reason).To(Equal("Succeeded"))
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

			sampleSecret := tb.Catalog.SecretWithoutAnnotation(buildObject.Spec.Output.SecretRef.Name, buildObject.Namespace)

			// generate resources
			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillValidation(buildName)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(buildObject.Status.Reason).To(Equal("Succeeded"))

			// delete a secret
			Expect(tb.DeleteSecret(buildObject.Spec.Output.SecretRef.Name)).To(BeNil())

			// assert that the validation happened one more time
			buildObject, err = tb.GetBuild(buildName)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
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
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(buildObject.Status.Reason).To(Equal(fmt.Sprintf("secret %s does not exist", buildObject.Spec.Output.SecretRef.Name)))

			sampleSecret := tb.Catalog.SecretWithoutAnnotation(buildObject.Spec.Output.SecretRef.Name, buildObject.Namespace)

			// generate resources
			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())

			// // assert that the validation happened one more time
			buildObject, err = tb.GetBuildTillRegistration(buildName, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(buildObject.Status.Reason).To(Equal(fmt.Sprintf("secret %s does not exist", buildObject.Spec.Output.SecretRef.Name)))

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
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(buildObject.Status.Reason).To(Equal(fmt.Sprintf("secret %s does not exist", "fake-secret")))

			sampleSecret := tb.Catalog.SecretWithoutAnnotation(buildObject.Spec.Output.SecretRef.Name, buildObject.Namespace)

			// generate resources
			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())
			// validate build status again
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(buildObject.Status.Reason).To(Equal(fmt.Sprintf("secret %s does not exist", "fake-secret")))

			// we modify the annotation so automatic delete does not take place
			data := []byte(fmt.Sprintf(`{"metadata":{"annotations":{"%s":"true"}}}`, v1alpha1.AnnotationBuildRefSecret))

			_, err = tb.PatchSecret(buildObject.Spec.Output.SecretRef.Name, data)
			Expect(err).To(BeNil())

			// // assert that the validation happened one more time
			buildObject, err = tb.GetBuildTillRegistration(buildName, corev1.ConditionTrue)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(buildObject.Status.Reason).To(Equal("Succeeded"))

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

			sampleSecret := tb.Catalog.SecretWithAnnotation(buildObject.Spec.Source.SecretRef.Name, buildObject.Namespace)

			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillValidation(buildName)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(buildObject.Status.Reason).To(Equal("Succeeded"))

			// delete a secret
			Expect(tb.DeleteSecret(buildObject.Spec.Source.SecretRef.Name)).To(BeNil())

			// assert that the validation happened one more time
			buildObject, err = tb.GetBuildTillRegistration(buildName, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(buildObject.Status.Reason).To(Equal(fmt.Sprintf("secret %s does not exist", buildObject.Spec.Source.SecretRef.Name)))

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
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(buildObject.Status.Reason).To(Equal(fmt.Sprintf("secret %s does not exist", buildObject.Spec.Source.SecretRef.Name)))

			sampleSecret := tb.Catalog.SecretWithAnnotation(buildObject.Spec.Source.SecretRef.Name, buildObject.Namespace)

			// generate resources
			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())

			// assert that the validation happened one more time
			buildObject, err = tb.GetBuildTillRegistration(buildName, corev1.ConditionTrue)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(buildObject.Status.Reason).To(Equal("Succeeded"))
		})
	})

	Context("when a build reference a secret with annotations for the spec builder", func() {
		It("should validate the Build after secret deletion", func() {

			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildWithBuilderRefSecret),
			)
			Expect(err).To(BeNil())

			sampleSecret := tb.Catalog.SecretWithAnnotation(buildObject.Spec.BuilderImage.SecretRef.Name, buildObject.Namespace)

			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillValidation(buildName)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(buildObject.Status.Reason).To(Equal("Succeeded"))

			// delete a secret
			Expect(tb.DeleteSecret(buildObject.Spec.BuilderImage.SecretRef.Name)).To(BeNil())

			// assert that the validation happened one more time
			buildObject, err = tb.GetBuildTillRegistration(buildName, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(buildObject.Status.Reason).To(Equal(fmt.Sprintf("secret %s does not exist", buildObject.Spec.BuilderImage.SecretRef.Name)))

		})

		It("should validate when a missing secret is recreated", func() {
			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildWithBuilderRefSecret),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillValidation(buildName)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(buildObject.Status.Reason).To(Equal(fmt.Sprintf("secret %s does not exist", buildObject.Spec.BuilderImage.SecretRef.Name)))

			sampleSecret := tb.Catalog.SecretWithAnnotation(buildObject.Spec.BuilderImage.SecretRef.Name, buildObject.Namespace)

			// generate resources
			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())

			// assert that the validation happened one more time
			buildObject, err = tb.GetBuildTillRegistration(buildName, corev1.ConditionTrue)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(buildObject.Status.Reason).To(Equal("Succeeded"))
		})
	})

	Context("when a build reference multiple secrets with annotations for a build instance", func() {
		It("should validate the Build after secret deletion", func() {

			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildWithMultipleRefSecrets),
			)
			Expect(err).To(BeNil())

			specSourceSecret := tb.Catalog.SecretWithAnnotation(buildObject.Spec.Source.SecretRef.Name, buildObject.Namespace)
			specBuilderSecret := tb.Catalog.SecretWithAnnotation(buildObject.Spec.BuilderImage.SecretRef.Name, buildObject.Namespace)

			Expect(tb.CreateSecret(specSourceSecret)).To(BeNil())
			Expect(tb.CreateSecret(specBuilderSecret)).To(BeNil())

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillValidation(buildName)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(buildObject.Status.Reason).To(Equal("Succeeded"))

			// delete a secret
			Expect(tb.DeleteSecret(specSourceSecret.Name)).To(BeNil())
			Expect(tb.DeleteSecret(specBuilderSecret.Name)).To(BeNil())

			buildObject, err = tb.GetBuildTillReasonContainsSubstring(buildName, "do not exist")
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(buildObject.Status.Reason).To(ContainSubstring(specSourceSecret.Name))
			Expect(buildObject.Status.Reason).To(ContainSubstring(specBuilderSecret.Name))

		})

		It("should validate when a missing secret is recreated", func() {
			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildWithMultipleRefSecrets),
			)
			Expect(err).To(BeNil())

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillValidation(buildName)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(buildObject.Status.Reason).To(ContainSubstring("do not exist"))
			Expect(buildObject.Status.Reason).To(ContainSubstring(buildObject.Spec.Source.SecretRef.Name))
			Expect(buildObject.Status.Reason).To(ContainSubstring(buildObject.Spec.BuilderImage.SecretRef.Name))

			specSourceSecret := tb.Catalog.SecretWithAnnotation(buildObject.Spec.Source.SecretRef.Name, buildObject.Namespace)
			specBuilderSecret := tb.Catalog.SecretWithAnnotation(buildObject.Spec.BuilderImage.SecretRef.Name, buildObject.Namespace)

			// generate resources
			Expect(tb.CreateSecret(specSourceSecret)).To(BeNil())
			Expect(tb.CreateSecret(specBuilderSecret)).To(BeNil())

			// assert that the validation happened one more time
			buildObject, err = tb.GetBuildTillRegistration(buildName, corev1.ConditionTrue)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(buildObject.Status.Reason).To(Equal("Succeeded"))
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

			specSourceSecret := tb.Catalog.SecretWithAnnotation(firstBuildObject.Spec.Source.SecretRef.Name, firstBuildObject.Namespace)

			Expect(tb.CreateSecret(specSourceSecret)).To(BeNil())

			Expect(tb.CreateBuild(firstBuildObject)).To(BeNil())
			Expect(tb.CreateBuild(secondBuildObject)).To(BeNil())

			// wait until the Build finish the validation
			o, err := tb.GetBuildTillValidation(firstBuildName)
			Expect(err).To(BeNil())
			Expect(o.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(o.Status.Reason).To(Equal("Succeeded"))

			o, err = tb.GetBuildTillValidation(secondBuildName)
			Expect(err).To(BeNil())
			Expect(o.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(o.Status.Reason).To(Equal("Succeeded"))

			// delete a secret
			Expect(tb.DeleteSecret(specSourceSecret.Name)).To(BeNil())

			// assert that the validation happened one more time
			o, err = tb.GetBuildTillRegistration(firstBuildName, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			Expect(o.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(o.Status.Reason).To(Equal(fmt.Sprintf("secret %s does not exist", firstBuildObject.Spec.Source.SecretRef.Name)))

			// assert that the validation happened one more time
			o, err = tb.GetBuildTillRegistration(secondBuildName, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			Expect(o.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(o.Status.Reason).To(Equal(fmt.Sprintf("secret %s does not exist", secondBuildObject.Spec.Source.SecretRef.Name)))
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
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))

			buildObject, err = tb.GetBuildTillValidation(secondBuildName)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))

			specSourceSecret := tb.Catalog.SecretWithAnnotation(firstBuildObject.Spec.Source.SecretRef.Name, firstBuildObject.Namespace)

			// generate resources
			Expect(tb.CreateSecret(specSourceSecret)).To(BeNil())

			// assert that the validation happened one more time
			o, err := tb.GetBuildTillRegistration(firstBuildName, corev1.ConditionTrue)
			Expect(err).To(BeNil())
			Expect(o.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(o.Status.Reason).To(Equal("Succeeded"))

			o, err = tb.GetBuildTillRegistration(secondBuildName, corev1.ConditionTrue)
			Expect(err).To(BeNil())
			Expect(o.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(o.Status.Reason).To(Equal("Succeeded"))
		})
	})
})

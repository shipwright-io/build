// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package integration_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	test "github.com/shipwright-io/build/test/v1beta1_samples"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/pointer"
)

var _ = Describe("Integration tests for retention limits and ttls for succeeded buildruns.", func() {
	var (
		cbsObject      *v1beta1.ClusterBuildStrategy
		buildObject    *v1beta1.Build
		buildRunObject *v1beta1.BuildRun
		buildSample    []byte
		buildRunSample []byte
	)

	// Load the ClusterBuildStrategies before each test case
	BeforeEach(func() {
		cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategyNoOp))
		Expect(err).To(BeNil())

		err = tb.CreateClusterBuildStrategy(cbsObject)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		_, err = tb.GetBuild(buildObject.Name)
		if err == nil {
			Expect(tb.DeleteBuild(buildObject.Name)).To(BeNil())
		}

		err := tb.DeleteClusterBuildStrategy(cbsObject.Name)
		Expect(err).To(BeNil())
	})

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

	Context("When a buildrun related to a build with short ttl set succeeds", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionTTLFive)
			buildRunSample = []byte(test.MinimalBuildRunRetention)
		})

		It("Should not find the buildrun after few seconds after it succeeds", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionTrue))
			_, err = tb.GetBRTillNotFound(buildRunObject.Name, time.Second*1, time.Second*15)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("When a buildrun with short ttl set succeeds. TTL field exists in the buildrun spec", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionLimitOne)
			buildRunSample = []byte(test.MinimalBuildRunRetentionTTLFive)
		})

		It("Should not find the buildrun few seconds after it succeeds", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionTrue))
			_, err = tb.GetBRTillNotFound(buildRunObject.Name, time.Second*1, time.Second*15)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("When a buildrun with short ttl set fails because buildref not found. TTL field exists in the buildrun spec", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionLimitOne)
			buildRunSample = []byte(test.MinimalBuildRunRetentionTTLFive)
		})

		It("Should not find the buildrun few seconds after it fails", func() {

			buildRunObject.Spec.Build.Name = pointer.String("non-existent-buildref")
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionFalse))
			_, err = tb.GetBRTillNotFound(buildRunObject.Name, time.Second*1, time.Second*15)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("When we lower a successfully completed buildrun's ttl retention parameters. Original param - 5m, Updated param - 5s", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionLimitOne)
			buildRunSample = []byte(test.MinimalBuildRunRetentionTTLFive)
		})

		It("Should not find the buildrun few seconds after it succeeds", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Make the TTL 5 minutes
			buildRunObject.Spec.Retention.TTLAfterSucceeded.Duration = time.Minute * 5
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionTrue))

			// Make the TTL 5 seconds
			br.Spec.Retention.TTLAfterSucceeded.Duration = time.Second * 5
			Expect(tb.UpdateBR(br)).To(BeNil())
			_, err = tb.GetBRTillNotFound(buildRunObject.Name, time.Second*1, time.Second*15)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("Multiple successful buildruns related to build with limit 1", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionLimitOne)
			buildRunSample = []byte(test.MinimalBuildRunRetention)
		})

		It("Should not find the older successful buildrun.", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Create first buildrun
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-1"
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			br1, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br1.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionTrue))
			// Create second buildrun
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-2"
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			br2, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br2.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionTrue))

			_, err = tb.GetBRTillNotFound(BUILDRUN+tb.Namespace+"-1", time.Second*1, time.Second*5)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
			_, err = tb.GetBRTillCompletion(BUILDRUN + tb.Namespace + "-2")
			Expect(err).To(BeNil())
		})

	})

	Context("When a buildrun that has TTL defined in its spec and in the corresponding build's spec succeeds", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionTTLOneMin)
			buildRunSample = []byte(test.MinimalBuildRunRetentionTTLFive)
		})

		It("Should honor the TTL defined in the buildrun.", func() {
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionTrue))
			_, err = tb.GetBRTillNotFound(buildRunObject.Name, time.Second*1, time.Second*15)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())

		})
	})

	Context("Multiple buildruns with different build limits for failure and success", func() {

		BeforeEach(func() {
			Expect(err).To(BeNil())
			buildSample = []byte(test.MinimalBuildWithRetentionLimitDiff)
			buildRunSample = []byte(test.MinimalBuildahBuildRunWithExitCode)
		})
		// Case with failedLimit 1 and succeededLimit 2.
		// It ensures that only relevant buildruns are affected by retention parameters
		It("Should not find the old failed buildrun if the limit has been triggered", func() {

			// Create build
			Expect(tb.CreateBuild(buildObject)).To(BeNil())
			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Create 2 successful buildruns
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-success-1"
			err = tb.CreateBR(buildRunObject)

			buildRunObject.Name = BUILDRUN + tb.Namespace + "-success-2"
			err = tb.CreateBR(buildRunObject)

			// Create 1 failed buildrun
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-fail-1"
			str := "false"
			falseParam := v1beta1.ParamValue{Name: "exit-command", SingleValue: &v1beta1.SingleValue{Value: &str}}
			buildRunObject.Spec.ParamValues = []v1beta1.ParamValue{falseParam}
			err = tb.CreateBR(buildRunObject)

			// Wait for buildrun completion
			br1, err := tb.GetBRTillCompletion(BUILDRUN + tb.Namespace + "-success-1")
			Expect(err).To(BeNil())
			Expect(br1.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionTrue))

			br2, err := tb.GetBRTillCompletion(BUILDRUN + tb.Namespace + "-success-2")
			Expect(err).To(BeNil())
			Expect(br2.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionTrue))

			br3, err := tb.GetBRTillCompletion(BUILDRUN + tb.Namespace + "-fail-1")
			Expect(err).To(BeNil())
			Expect(br3.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionFalse))

			// Create 1 failed buildrun.
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-fail-2"
			buildRunObject.Spec.ParamValues = []v1beta1.ParamValue{falseParam}
			err = tb.CreateBR(buildRunObject)
			Expect(err).To(BeNil())
			br4, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br4.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionFalse))

			// Check that the older failed buildrun has been deleted while the successful buildruns exist
			_, err = tb.GetBRTillNotFound(BUILDRUN+tb.Namespace+"-fail-1", time.Second*1, time.Second*5)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
			_, err = tb.GetBRTillCompletion(BUILDRUN + tb.Namespace + "-success-1")
			Expect(err).To(BeNil())
			_, err = tb.GetBRTillCompletion(BUILDRUN + tb.Namespace + "-success-2")
			Expect(err).To(BeNil())
		})
	})

})

var _ = Describe("Integration tests for retention limits and ttls of buildRuns that fail", func() {
	var (
		cbsObject      *v1beta1.ClusterBuildStrategy
		buildObject    *v1beta1.Build
		buildRunObject *v1beta1.BuildRun
		buildSample    []byte
		buildRunSample []byte
	)
	// Load the ClusterBuildStrategies before each test case
	BeforeEach(func() {
		cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategyNoOp))
		Expect(err).To(BeNil())

		err = tb.CreateClusterBuildStrategy(cbsObject)
		Expect(err).To(BeNil())
	})
	AfterEach(func() {

		_, err = tb.GetBuild(buildObject.Name)
		if err == nil {
			Expect(tb.DeleteBuild(buildObject.Name)).To(BeNil())
		}

		err := tb.DeleteClusterBuildStrategy(cbsObject.Name)
		Expect(err).To(BeNil())
	})

	JustBeforeEach(func() {
		if buildSample != nil {
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, STRATEGY+tb.Namespace, buildSample)
			Expect(err).To(BeNil())
		}

		if buildRunSample != nil {
			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, BUILD+tb.Namespace, buildRunSample)
			Expect(err).To(BeNil())
			str := "false"
			falseParam := v1beta1.ParamValue{Name: "exit-command", SingleValue: &v1beta1.SingleValue{Value: &str}}
			buildRunObject.Spec.ParamValues = []v1beta1.ParamValue{falseParam}
		}
	})

	Context("When a buildrun related to a build with short ttl set fails", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionTTLFive)
			buildRunSample = []byte(test.MinimalBuildRunRetention)
		})

		It("Should not find the buildrun few seconds after it fails", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionFalse))
			_, err = tb.GetBRTillNotFound(buildRunObject.Name, time.Second*1, time.Second*15)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("When we lower a failed completed buildrun's ttl retention parameters. Original param - 5m, Updated param - 5s", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionLimitOne)
			buildRunSample = []byte(test.MinimalBuildRunRetentionTTLFive)
		})

		It("Should not find the buildrun few seconds after it succeeds", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Make the TTL 5 minutes
			buildRunObject.Spec.Retention.TTLAfterFailed.Duration = time.Minute * 5
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionFalse))

			// Make the TTL 5 seconds
			br.Spec.Retention.TTLAfterFailed.Duration = time.Second * 5
			Expect(tb.UpdateBR(br)).To(BeNil())
			_, err = tb.GetBRTillNotFound(buildRunObject.Name, time.Second*1, time.Second*15)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("When a buildrun with short ttl set fails. TTL is defined in the buildrun specs", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionLimitOne)
			buildRunSample = []byte(test.MinimalBuildRunRetentionTTLFive)
		})

		It("Should not find the buildrun few seconds after it fails", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionFalse))
			_, err = tb.GetBRTillNotFound(buildRunObject.Name, time.Second*1, time.Second*15)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("When a buildrun that has TTL defined in its spec and on the corresponding build's spec fails", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionTTLOneMin)
			buildRunSample = []byte(test.MinimalBuildRunRetentionTTLFive)
		})

		It("Should honor the TTL defined in the buildrun.", func() {
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionFalse))
			_, err = tb.GetBRTillNotFound(buildRunObject.Name, time.Second*1, time.Second*15)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())

		})
	})

	Context("Multiple failed buildruns related to build with limit 1", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionLimitOne)
			buildRunSample = []byte(test.MinimalBuildRunRetention)
		})

		It("Should not find the older failed buildrun", func() {
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Create first buildrun
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-1"
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			br1, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br1.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionFalse))
			// Create second buildrun
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-2"
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			br2, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br2.Status.GetCondition(v1beta1.Succeeded).Status).To(Equal(corev1.ConditionFalse))

			_, err = tb.GetBRTillNotFound(BUILDRUN+tb.Namespace+"-1", time.Second*1, time.Second*5)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
			_, err = tb.GetBRTillCompletion(BUILDRUN + tb.Namespace + "-2")
			Expect(err).To(BeNil())
		})

	})
})

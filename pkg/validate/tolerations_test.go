// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	corev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/ptr"

	. "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/validate"
)

var _ = Describe("ValidateTolerations", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.TODO()
	})

	var validate = func(build *Build) {
		GinkgoHelper()

		var validator = &validate.TolerationsRef{Build: build}
		Expect(validator.ValidatePath(ctx)).To(Succeed())
	}

	var sampleBuild = func(toleration v1.Toleration) *Build {
		return &Build{
			ObjectMeta: corev1.ObjectMeta{
				Namespace: "foo",
				Name:      "bar",
			},
			Spec: BuildSpec{
				Tolerations: []v1.Toleration{toleration},
			},
		}
	}

	Context("when tolerations is specified", func() {
		It("should fail an empty key and empty value", func() {
			build := sampleBuild(v1.Toleration{Key: "", Value: "", Operator: v1.TolerationOpEqual, Effect: v1.TaintEffectNoSchedule})
			validate(build)
			Expect(*build.Status.Reason).To(Equal(TolerationNotValid))
			Expect(*build.Status.Message).To(ContainSubstring(validation.EmptyError()))
		})

		It("should pass a valid key and valid value", func() {
			build := sampleBuild(v1.Toleration{Key: "validkey", Value: "validvalue", Operator: v1.TolerationOpEqual, Effect: v1.TaintEffectNoSchedule})
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should pass a valid key and empty value", func() {
			build := sampleBuild(v1.Toleration{Key: "validkey", Value: "", Operator: v1.TolerationOpEqual, Effect: v1.TaintEffectNoSchedule})
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should fail an invalid key and empty value", func() {
			build := sampleBuild(v1.Toleration{Key: "invalidkey!", Value: "", Operator: v1.TolerationOpEqual, Effect: v1.TaintEffectNoSchedule})
			validate(build)
			Expect(*build.Status.Reason).To(Equal(TolerationNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("Toleration key not valid"))
		})

		It("should fail an invalid key and invalid value", func() {
			build := sampleBuild(v1.Toleration{Key: "invalidkey!", Value: "invalidvalue!", Operator: v1.TolerationOpEqual, Effect: v1.TaintEffectNoSchedule})
			validate(build)
			Expect(*build.Status.Reason).To(Equal(TolerationNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("Toleration key not valid"))
		})

		It("should fail a valid key and invalid value", func() {
			build := sampleBuild(v1.Toleration{Key: "validkey", Value: "invalidvalue!", Operator: v1.TolerationOpEqual, Effect: v1.TaintEffectNoSchedule})
			validate(build)
			Expect(*build.Status.Reason).To(Equal(TolerationNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("Toleration value not valid"))
		})

		It("should fail an invalid operator", func() {
			build := sampleBuild(v1.Toleration{Key: "validkey", Value: "validvalue", Operator: "invalidoperator", Effect: v1.TaintEffectNoSchedule})
			validate(build)
			Expect(*build.Status.Reason).To(Equal(TolerationNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("Toleration operator not valid"))
		})

		It("should pass an empty taint effect", func() {
			build := sampleBuild(v1.Toleration{Key: "validkey", Value: "validvalue", Operator: v1.TolerationOpEqual, Effect: ""})
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should pass a taint effect of NoSchedule", func() {
			build := sampleBuild(v1.Toleration{Key: "validkey", Value: "validvalue", Operator: v1.TolerationOpEqual, Effect: v1.TaintEffectNoSchedule})
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should fail an invalid taint effect", func() {
			build := sampleBuild(v1.Toleration{Key: "validkey", Value: "validvalue", Operator: v1.TolerationOpEqual, Effect: v1.TaintEffectNoExecute})
			validate(build)
			Expect(*build.Status.Reason).To(Equal(TolerationNotValid))
			Expect(*build.Status.Message).To(ContainSubstring(fmt.Sprintf("Only the '%v' toleration effect is supported.", v1.TaintEffectNoSchedule)))
		})

		It("should fail specifying tolerationSeconds", func() {
			build := sampleBuild(v1.Toleration{Key: "validkey", Value: "validvalue", Operator: v1.TolerationOpEqual, Effect: v1.TaintEffectNoSchedule, TolerationSeconds: ptr.To(int64(10))})
			validate(build)
			Expect(*build.Status.Reason).To(Equal(TolerationNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("Specifying TolerationSeconds is not supported"))
		})
	})
})

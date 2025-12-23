// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/validate"
)

var _ = Describe("ValidateRuntimeClassName", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.TODO()
	})

	var validate = func(build *Build) {
		GinkgoHelper()

		var validator = &validate.RuntimeClassNameRef{Build: build}
		Expect(validator.ValidatePath(ctx)).To(Succeed())
	}

	var sampleBuild = func(runtimeClassName string) *Build {
		return &Build{
			ObjectMeta: corev1.ObjectMeta{
				Namespace: "foo",
				Name:      "bar",
			},
			Spec: BuildSpec{
				RuntimeClassName: &runtimeClassName,
			},
		}
	}

	Context("when runtimeClassName is specified", func() {
		It("should fail an empty name", func() {
			build := sampleBuild("")
			validate(build)
			Expect(*build.Status.Reason).To(Equal(RuntimeClassNameNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("RuntimeClassName not valid"))
		})

		It("should fail an invalid name with uppercase", func() {
			build := sampleBuild("InvalidName")
			validate(build)
			Expect(*build.Status.Reason).To(Equal(RuntimeClassNameNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("RuntimeClassName not valid"))
		})

		It("should fail an invalid name with special characters", func() {
			build := sampleBuild("invalid_name!")
			validate(build)
			Expect(*build.Status.Reason).To(Equal(RuntimeClassNameNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("RuntimeClassName not valid"))
		})

		It("should pass a valid name", func() {
			build := sampleBuild("kata-containers")
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should pass a valid name with dots", func() {
			build := sampleBuild("gvisor.runsc")
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})
	})
})

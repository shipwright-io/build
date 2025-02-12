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

var _ = Describe("ValidateNodeSelector", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.TODO()
	})

	var validate = func(build *Build) {
		GinkgoHelper()

		var validator = &validate.NodeSelectorRef{Build: build}
		Expect(validator.ValidatePath(ctx)).To(Succeed())
	}

	var sampleBuild = func(key string, value string) *Build {
		return &Build{
			ObjectMeta: corev1.ObjectMeta{
				Namespace: "foo",
				Name:      "bar",
			},
			Spec: BuildSpec{
				NodeSelector: map[string]string{key: value},
			},
		}
	}

	Context("when node selector is specified", func() {
		It("should fail an empty key and value", func() {
			build := sampleBuild("", "")
			validate(build)
			Expect(*build.Status.Reason).To(Equal(NodeSelectorNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("Node selector key not valid"))
		})

		It("should fail an empty key and valid value", func() {
			build := sampleBuild("", "validvalue")
			validate(build)
			Expect(*build.Status.Reason).To(Equal(NodeSelectorNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("Node selector key not valid"))
		})

		It("should fail an empty key and invalid value", func() {
			build := sampleBuild("", "invalidvalue!")
			validate(build)
			Expect(*build.Status.Reason).To(Equal(NodeSelectorNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("Node selector key not valid"))
		})

		It("should pass a valid key and valid value", func() {
			build := sampleBuild("validkey", "validvalue")
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should fail a valid key and invalid value", func() {
			build := sampleBuild("validkey", "invalidvalue!")
			validate(build)
			Expect(*build.Status.Reason).To(Equal(NodeSelectorNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("Node selector value not valid"))
		})

		It("should pass a valid key and empty value", func() {
			build := sampleBuild("validkey", "")
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should fail an invalid key and empty value", func() {
			build := sampleBuild("invalidkey!", "")
			validate(build)
			Expect(*build.Status.Reason).To(Equal(NodeSelectorNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("Node selector key not valid"))
		})

		It("should fail an invalid key and valid value", func() {
			build := sampleBuild("invalidkey!", "validvalue")
			validate(build)
			Expect(*build.Status.Reason).To(Equal(NodeSelectorNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("Node selector key not valid"))
		})

		It("should fail both an invalid key and invalid value", func() {
			build := sampleBuild("invalidkey!", "invalidvalue!")
			validate(build)
			Expect(*build.Status.Reason).To(Equal(NodeSelectorNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("Node selector key not valid"))
		})
	})
})

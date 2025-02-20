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

var _ = Describe("ValidateSchedulerName", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.TODO()
	})

	var validate = func(build *Build) {
		GinkgoHelper()

		var validator = &validate.SchedulerNameRef{Build: build}
		Expect(validator.ValidatePath(ctx)).To(Succeed())
	}

	var sampleBuild = func(schedulerName string) *Build {
		return &Build{
			ObjectMeta: corev1.ObjectMeta{
				Namespace: "foo",
				Name:      "bar",
			},
			Spec: BuildSpec{
				SchedulerName: &schedulerName,
			},
		}
	}

	Context("when schedulerName is specified", func() {
		It("should fail an empty name", func() {
			build := sampleBuild("")
			validate(build)
			Expect(*build.Status.Reason).To(Equal(SchedulerNameNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("Scheduler name not valid"))
		})

		It("should fail an invalid name", func() {
			build := sampleBuild("invalidname!")
			validate(build)
			Expect(*build.Status.Reason).To(Equal(SchedulerNameNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("Scheduler name not valid"))
		})

		It("should pass a valid name", func() {
			build := sampleBuild("validname")
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})
	})
})

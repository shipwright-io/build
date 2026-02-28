// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/validate"
)

var _ = Describe("BuildRunStepResources", func() {
	var (
		strategySteps         []buildv1beta1.Step
		buildRunStepResources []buildv1beta1.StepResourceOverride
	)

	BeforeEach(func() {
		strategySteps = []buildv1beta1.Step{
			{Name: "step-build"},
			{Name: "step-push"},
			{Name: "step-prepare"},
		}
		buildRunStepResources = nil
	})

	Context("when buildRun stepResources is nil or empty", func() {
		It("should return valid for nil or empty stepResources", func() {
			// Test nil
			valid, reason, message := validate.BuildRunStepResources(strategySteps, nil)
			Expect(valid).To(BeTrue())
			Expect(reason).To(BeEmpty())
			Expect(message).To(BeEmpty())

			// Test empty slice (same code path as nil in Go)
			valid, reason, message = validate.BuildRunStepResources(strategySteps, []buildv1beta1.StepResourceOverride{})
			Expect(valid).To(BeTrue())
			Expect(reason).To(BeEmpty())
			Expect(message).To(BeEmpty())
		})
	})

	Context("when buildRun stepResources references valid steps", func() {
		It("should return valid for a single valid step", func() {
			buildRunStepResources = []buildv1beta1.StepResourceOverride{
				{
					Name: "step-build",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
					},
				},
			}
			valid, reason, message := validate.BuildRunStepResources(strategySteps, buildRunStepResources)
			Expect(valid).To(BeTrue())
			Expect(reason).To(BeEmpty())
			Expect(message).To(BeEmpty())
		})

	})

	Context("when buildRun stepResources references invalid steps", func() {
		It("should return invalid when one of multiple steps is invalid", func() {
			buildRunStepResources = []buildv1beta1.StepResourceOverride{
				{
					Name: "step-build",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("500m"),
						},
					},
				},
				{
					Name: "non-existent-step",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
					},
				},
			}
			valid, reason, message := validate.BuildRunStepResources(strategySteps, buildRunStepResources)
			Expect(valid).To(BeFalse())
			Expect(reason).To(Equal(string(buildv1beta1.UndefinedStepResource)))
			Expect(message).To(ContainSubstring("non-existent-step"))
		})
	})

	Context("when strategy has no steps", func() {
		It("should return invalid for any stepResources", func() {
			strategySteps = []buildv1beta1.Step{}
			buildRunStepResources = []buildv1beta1.StepResourceOverride{
				{
					Name: "step-build",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("500m"),
						},
					},
				},
			}
			valid, reason, _ := validate.BuildRunStepResources(strategySteps, buildRunStepResources)
			Expect(valid).To(BeFalse())
			Expect(reason).To(Equal(string(buildv1beta1.UndefinedStepResource)))
		})
	})
})

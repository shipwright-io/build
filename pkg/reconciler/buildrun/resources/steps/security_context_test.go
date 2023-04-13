// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package steps_test

import (
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/steps"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UpdateSecurityContext", func() {

	var buildStrategySteps []buildapi.BuildStep
	var tektonStep *tektonapi.Step

	JustBeforeEach(func() {
		steps.UpdateSecurityContext(tektonStep, buildStrategySteps)
	})

	Context("for build strategy steps that don't use a common runAsUser and runAsGroup", func() {

		BeforeEach(func() {
			buildStrategySteps = []buildapi.BuildStep{{
				Container: corev1.Container{
					Name: "first-step",
					SecurityContext: &corev1.SecurityContext{
						RunAsUser:  pointer.Int64(891),
						RunAsGroup: pointer.Int64(1210),
					},
				},
			}, {
				Container: corev1.Container{
					Name: "second-step",
					SecurityContext: &corev1.SecurityContext{
						RunAsUser: pointer.Int64(891),
					},
				},
			}}
		})

		Context("for a Tekton Step with a security context", func() {

			BeforeEach(func() {
				tektonStep = &tektonapi.Step{
					Name: "step",
					SecurityContext: &corev1.SecurityContext{
						RunAsUser:  pointer.Int64(1001),
						RunAsGroup: pointer.Int64(1000),
					},
				}
			})

			It("retains the security context", func() {
				Expect(tektonStep.SecurityContext.RunAsUser).To(Equal(pointer.Int64(1001)))
				Expect(tektonStep.SecurityContext.RunAsGroup).To(Equal(pointer.Int64(1000)))
			})
		})

		Context("for a Tekton Step without a security context", func() {

			BeforeEach(func() {
				tektonStep = &tektonapi.Step{
					Name: "step",
				}
			})

			It("does not introduce the security context", func() {
				Expect(tektonStep.SecurityContext).To(BeNil())
			})
		})
	})

	Context("for build strategy steps that use a common runAsUser and runAsGroup", func() {

		BeforeEach(func() {
			buildStrategySteps = []buildapi.BuildStep{{
				Container: corev1.Container{
					Name: "first-step",
					SecurityContext: &corev1.SecurityContext{
						RunAsUser:  pointer.Int64(891),
						RunAsGroup: pointer.Int64(1210),
					},
				},
			}, {
				Container: corev1.Container{
					Name: "second-step",
					SecurityContext: &corev1.SecurityContext{
						RunAsUser:  pointer.Int64(891),
						RunAsGroup: pointer.Int64(1210),
					},
				},
			}}
		})

		Context("for a Tekton Step with a security context", func() {

			BeforeEach(func() {
				tektonStep = &tektonapi.Step{
					Name: "step",
					SecurityContext: &corev1.SecurityContext{
						RunAsUser:  pointer.Int64(1001),
						RunAsGroup: pointer.Int64(1000),
					},
				}
			})

			It("updates the security context", func() {
				Expect(tektonStep.SecurityContext.RunAsUser).To(Equal(pointer.Int64(891)))
				Expect(tektonStep.SecurityContext.RunAsGroup).To(Equal(pointer.Int64(1210)))
			})
		})

		Context("for a Tekton Step without a security context", func() {

			BeforeEach(func() {
				tektonStep = &tektonapi.Step{
					Name: "step",
				}
			})

			It("introduces the security context", func() {
				Expect(tektonStep.SecurityContext).ToNot(BeNil())
				Expect(tektonStep.SecurityContext.RunAsUser).To(Equal(pointer.Int64(891)))
				Expect(tektonStep.SecurityContext.RunAsGroup).To(Equal(pointer.Int64(1210)))
			})
		})
	})
})

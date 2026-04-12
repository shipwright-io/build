// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/validate"
)

var _ = Describe("BuildSpecOutputValidator", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.TODO()
	})

	var validate = func(build *buildapi.Build) {
		GinkgoHelper()

		var validator = &validate.BuildSpecOutputValidator{Build: build}
		Expect(validator.ValidatePath(ctx)).To(Succeed())
	}

	Context("output timestamp is specified", func() {
		var sampleBuild = func(timestamp string) *buildapi.Build {
			return &buildapi.Build{
				ObjectMeta: corev1.ObjectMeta{
					Namespace: "foo",
					Name:      "bar",
				},
				Spec: buildapi.BuildSpec{
					Source: &buildapi.Source{
						Type: buildapi.GitType,
						Git: &buildapi.Git{
							URL: "https://github.com/shipwright-io/sample-go",
						},
					},
					Strategy: buildapi.Strategy{
						Name: "magic",
					},
					Output: buildapi.Image{
						Timestamp: &timestamp,
					},
				},
			}
		}

		It("should pass an empty string", func() {
			build := sampleBuild("")
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should pass with string Zero", func() {
			build := sampleBuild(buildapi.OutputImageZeroTimestamp)
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should pass with string SourceTimestamp", func() {
			build := sampleBuild(buildapi.OutputImageSourceTimestamp)
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should pass with string BuildTimestamp", func() {
			build := sampleBuild(buildapi.OutputImageBuildTimestamp)
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should fail with string SourceTimestamp in case there are no sources", func() {
			build := sampleBuild(buildapi.OutputImageSourceTimestamp)
			build.Spec.Source = nil

			validate(build)
			Expect(*build.Status.Reason).To(Equal(buildapi.OutputTimestampNotSupported))
			Expect(*build.Status.Message).To(ContainSubstring("cannot use SourceTimestamp"))
		})

		It("should fail when invalid timestamp is used", func() {
			build := sampleBuild("WrongValue")

			validate(build)
			Expect(*build.Status.Reason).To(Equal(buildapi.OutputTimestampNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("output timestamp value is invalid"))
		})
	})
})

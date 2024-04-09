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

var _ = Describe("BuildSpecOutputValidator", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.TODO()
	})

	var validate = func(build *Build) {
		GinkgoHelper()

		var validator = &validate.BuildSpecOutputValidator{Build: build}
		Expect(validator.ValidatePath(ctx)).To(Succeed())
	}

	Context("output timestamp is specified", func() {
		var sampleBuild = func(timestamp string) *Build {
			return &Build{
				ObjectMeta: corev1.ObjectMeta{
					Namespace: "foo",
					Name:      "bar",
				},
				Spec: BuildSpec{
					Source: &Source{
						Type: GitType,
						Git: &Git{
							URL: "https://github.com/shipwright-io/sample-go",
						},
					},
					Strategy: Strategy{
						Name: "magic",
					},
					Output: Image{
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
			build := sampleBuild(OutputImageZeroTimestamp)
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should pass with string SourceTimestamp", func() {
			build := sampleBuild(OutputImageSourceTimestamp)
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should pass with string BuildTimestamp", func() {
			build := sampleBuild(OutputImageBuildTimestamp)
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should fail with string SourceTimestamp in case there are no sources", func() {
			build := sampleBuild(OutputImageSourceTimestamp)
			build.Spec.Source = nil

			validate(build)
			Expect(*build.Status.Reason).To(Equal(OutputTimestampNotSupported))
			Expect(*build.Status.Message).To(ContainSubstring("cannot use SourceTimestamp"))
		})

		It("should fail when invalid timestamp is used", func() {
			build := sampleBuild("WrongValue")

			validate(build)
			Expect(*build.Status.Reason).To(Equal(OutputTimestampNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("output timestamp value is invalid"))
		})
	})

	Context("output vulnerabilityScan is specified", func() {
		var sampleBuild = func(vulnerabilitySettings VulnerabilityScanOptions) *Build {
			return &Build{
				ObjectMeta: corev1.ObjectMeta{
					Namespace: "foo",
					Name:      "bar",
				},
				Spec: BuildSpec{
					Source: &Source{
						Type: GitType,
						Git: &Git{
							URL: "https://github.com/shipwright-io/sample-go",
						},
					},
					Strategy: Strategy{
						Name: "magic",
					},
					Output: Image{
						VulnerabilityScan: &vulnerabilitySettings,
					},
				},
			}
		}

		It("should pass for valid severities", func() {
			severites := []string{"low", "high", "medium"}
			for _, severity := range severites {
				sev := severity
				vulnerabilitySettings := VulnerabilityScanOptions{
					Ignore: &VulnerabilityIgnoreOptions{
						Severity: &sev,
					},
				}
				build := sampleBuild(vulnerabilitySettings)
				validate(build)
				Expect(build.Status.Reason).To(BeNil())
				Expect(build.Status.Message).To(BeNil())
			}
		})

		It("should fail for invvalid severities", func() {
			severities := "LOWE"
			vulnerabilitySettings := VulnerabilityScanOptions{
				Ignore: &VulnerabilityIgnoreOptions{
					Severity: &severities,
				},
			}
			build := sampleBuild(vulnerabilitySettings)
			validate(build)
			Expect(*build.Status.Reason).To(Equal(VulnerabilityScanSeverityNotValid))
			Expect(*build.Status.Message).To(ContainSubstring("vulnerability scan severity is invalid"))
		})
	})
})

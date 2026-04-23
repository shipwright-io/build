// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
				ObjectMeta: metav1.ObjectMeta{
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

	Context("output platforms is specified", func() {
		var buildWithOutputPlatforms = func(platforms []buildapi.ImagePlatform, nodeSelector map[string]string) *buildapi.Build {
			return &buildapi.Build{
				ObjectMeta: metav1.ObjectMeta{
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
						Name: "buildah",
					},
					Output: buildapi.Image{
						Image:     "quay.io/example/app:latest",
						Platforms: platforms,
					},
					NodeSelector: nodeSelector,
				},
			}
		}

		It("should pass with valid platforms", func() {
			build := buildWithOutputPlatforms([]buildapi.ImagePlatform{
				{OS: "linux", Arch: "amd64"},
				{OS: "linux", Arch: "arm64"},
			}, nil)
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should not apply platform validations when platforms is empty", func() {
			build := buildWithOutputPlatforms([]buildapi.ImagePlatform{}, nil)
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should fail when os is empty", func() {
			build := buildWithOutputPlatforms([]buildapi.ImagePlatform{
				{OS: "", Arch: "amd64"},
			}, nil)
			validate(build)
			Expect(*build.Status.Reason).To(Equal(buildapi.InvalidPlatform))
			Expect(*build.Status.Message).To(ContainSubstring("must specify both os and arch"))
		})

		It("should fail when arch is empty", func() {
			build := buildWithOutputPlatforms([]buildapi.ImagePlatform{
				{OS: "linux", Arch: ""},
			}, nil)
			validate(build)
			Expect(*build.Status.Reason).To(Equal(buildapi.InvalidPlatform))
			Expect(*build.Status.Message).To(ContainSubstring("must specify both os and arch"))
		})

		It("should fail with duplicate platform entries", func() {
			build := buildWithOutputPlatforms([]buildapi.ImagePlatform{
				{OS: "linux", Arch: "amd64"},
				{OS: "linux", Arch: "arm64"},
				{OS: "linux", Arch: "amd64"},
			}, nil)
			validate(build)
			Expect(*build.Status.Reason).To(Equal(buildapi.InvalidPlatform))
			Expect(*build.Status.Message).To(ContainSubstring("duplicate"))
			Expect(*build.Status.Message).To(ContainSubstring("linux/amd64"))
		})

		It("should fail when os is not a valid label-style value", func() {
			build := buildWithOutputPlatforms([]buildapi.ImagePlatform{
				{OS: "Linux", Arch: "amd64"},
			}, nil)
			validate(build)
			Expect(*build.Status.Reason).To(Equal(buildapi.InvalidPlatform))
			Expect(*build.Status.Message).To(ContainSubstring(`kubernetes.io/os`))
		})

		It("should fail when arch is not lowercase (not a real label value)", func() {
			build := buildWithOutputPlatforms([]buildapi.ImagePlatform{
				{OS: "linux", Arch: "AMD64"},
			}, nil)
			validate(build)
			Expect(*build.Status.Reason).To(Equal(buildapi.InvalidPlatform))
			Expect(*build.Status.Message).To(ContainSubstring(`kubernetes.io/arch`))
		})

		It("should fail when nodeSelector contains kubernetes.io/os", func() {
			build := buildWithOutputPlatforms([]buildapi.ImagePlatform{
				{OS: "linux", Arch: "amd64"},
			}, map[string]string{
				"kubernetes.io/os": "linux",
			})
			validate(build)
			Expect(*build.Status.Reason).To(Equal(buildapi.NodeSelectorPlatformConflict))
			Expect(*build.Status.Message).To(ContainSubstring("kubernetes.io/os"))
		})

		It("should fail when nodeSelector contains kubernetes.io/arch", func() {
			build := buildWithOutputPlatforms([]buildapi.ImagePlatform{
				{OS: "linux", Arch: "amd64"},
			}, map[string]string{
				"kubernetes.io/arch": "amd64",
			})
			validate(build)
			Expect(*build.Status.Reason).To(Equal(buildapi.NodeSelectorPlatformConflict))
			Expect(*build.Status.Message).To(ContainSubstring("kubernetes.io/arch"))
		})

		It("should pass when nodeSelector has unrelated labels", func() {
			build := buildWithOutputPlatforms([]buildapi.ImagePlatform{
				{OS: "linux", Arch: "amd64"},
			}, map[string]string{
				"disktype": "ssd",
			})
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})
	})
})

var _ = Describe("ValidateNodeAvailability", func() {
	readyNode := func(name, os, arch string) corev1.Node {
		return corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				Labels: map[string]string{
					corev1.LabelOSStable:   os,
					corev1.LabelArchStable: arch,
				},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		}
	}

	It("should pass when nodes exist for all platforms", func() {
		nodes := []corev1.Node{
			readyNode("node-amd64", "linux", "amd64"),
			readyNode("node-arm64", "linux", "arm64"),
		}
		platforms := []buildapi.ImagePlatform{
			{OS: "linux", Arch: "amd64"},
			{OS: "linux", Arch: "arm64"},
		}
		valid, reason, msg := validate.ValidateNodeAvailability(platforms, nodes)
		Expect(valid).To(BeTrue())
		Expect(reason).To(BeEmpty())
		Expect(msg).To(BeEmpty())
	})

	It("should fail when no node exists for a platform", func() {
		nodes := []corev1.Node{
			readyNode("node-amd64", "linux", "amd64"),
		}
		platforms := []buildapi.ImagePlatform{
			{OS: "linux", Arch: "amd64"},
			{OS: "linux", Arch: "s390x"},
		}
		valid, reason, msg := validate.ValidateNodeAvailability(platforms, nodes)
		Expect(valid).To(BeFalse())
		Expect(reason).To(Equal(string(buildapi.NodePlatformNotFound)))
		Expect(msg).To(ContainSubstring("linux/s390x"))
	})

	// tests availablePlatforms function
	It("should skip unschedulable nodes", func() {
		unschedulable := readyNode("node-amd64", "linux", "amd64")
		unschedulable.Spec.Unschedulable = true
		nodes := []corev1.Node{unschedulable}
		platforms := []buildapi.ImagePlatform{{OS: "linux", Arch: "amd64"}}
		valid, _, msg := validate.ValidateNodeAvailability(platforms, nodes)
		Expect(valid).To(BeFalse())
		Expect(msg).To(ContainSubstring("linux/amd64"))
	})

	It("should skip nodes that are not Ready", func() {
		notReady := corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-amd64",
				Labels: map[string]string{
					corev1.LabelOSStable:   "linux",
					corev1.LabelArchStable: "amd64",
				},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
				},
			},
		}
		platforms := []buildapi.ImagePlatform{{OS: "linux", Arch: "amd64"}}
		valid, _, msg := validate.ValidateNodeAvailability(platforms, []corev1.Node{notReady})
		Expect(valid).To(BeFalse())
		Expect(msg).To(ContainSubstring("linux/amd64"))
	})
})

var _ = Describe("ValidateMultiArchPreflight", func() {
	It("succeeds when executor is PipelineRun and platforms and nodeSelector are valid", func() {
		platforms := []buildapi.ImagePlatform{
			{OS: "linux", Arch: "amd64"},
			{OS: "linux", Arch: "arm64"},
		}
		valid, reason, msg := validate.ValidateMultiArchPreflight(platforms, map[string]string{"disktype": "ssd"}, "PipelineRun")
		Expect(valid).To(BeTrue())
		Expect(reason).To(BeEmpty())
		Expect(msg).To(BeEmpty())
	})

	It("returns ExecutorNotPipelineRun when executor is TaskRun", func() {
		platforms := []buildapi.ImagePlatform{{OS: "linux", Arch: "amd64"}}
		valid, reason, msg := validate.ValidateMultiArchPreflight(platforms, nil, "TaskRun")
		Expect(valid).To(BeFalse())
		Expect(reason).To(Equal(string(buildapi.ExecutorNotPipelineRun)))
		Expect(msg).To(ContainSubstring("PipelineRun executor mode"))
		Expect(msg).To(ContainSubstring("TaskRun"))
	})
})

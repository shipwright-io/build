// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/controller/fakes"
	"github.com/shipwright-io/build/pkg/validate"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
				ObjectMeta: metav1.ObjectMeta{
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

	Context("output multiArch is specified", func() {
		var multiArchBuild = func(multiArch *MultiArch, nodeSelector map[string]string) *Build {
			return &Build{
				ObjectMeta: metav1.ObjectMeta{
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
						Name: "buildah",
					},
					Output: Image{
						Image:     "quay.io/example/app:latest",
						MultiArch: multiArch,
					},
					NodeSelector: nodeSelector,
				},
			}
		}

		It("should pass with valid platforms", func() {
			build := multiArchBuild(&MultiArch{
				Platforms: []ImagePlatform{
					{OS: "linux", Arch: "amd64"},
					{OS: "linux", Arch: "arm64"},
				},
			}, nil)
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should fail with empty platforms list", func() {
			build := multiArchBuild(&MultiArch{
				Platforms: []ImagePlatform{},
			}, nil)
			validate(build)
			Expect(*build.Status.Reason).To(Equal(MultiArchInvalidPlatform))
			Expect(*build.Status.Message).To(ContainSubstring("at least one platform"))
		})

		It("should fail when os is empty", func() {
			build := multiArchBuild(&MultiArch{
				Platforms: []ImagePlatform{
					{OS: "", Arch: "amd64"},
				},
			}, nil)
			validate(build)
			Expect(*build.Status.Reason).To(Equal(MultiArchInvalidPlatform))
			Expect(*build.Status.Message).To(ContainSubstring("must specify both os and arch"))
		})

		It("should fail when arch is empty", func() {
			build := multiArchBuild(&MultiArch{
				Platforms: []ImagePlatform{
					{OS: "linux", Arch: ""},
				},
			}, nil)
			validate(build)
			Expect(*build.Status.Reason).To(Equal(MultiArchInvalidPlatform))
			Expect(*build.Status.Message).To(ContainSubstring("must specify both os and arch"))
		})

		It("should fail with duplicate platform entries", func() {
			build := multiArchBuild(&MultiArch{
				Platforms: []ImagePlatform{
					{OS: "linux", Arch: "amd64"},
					{OS: "linux", Arch: "arm64"},
					{OS: "linux", Arch: "amd64"},
				},
			}, nil)
			validate(build)
			Expect(*build.Status.Reason).To(Equal(MultiArchInvalidPlatform))
			Expect(*build.Status.Message).To(ContainSubstring("duplicate"))
			Expect(*build.Status.Message).To(ContainSubstring("linux/amd64"))
		})

		It("should fail when nodeSelector contains kubernetes.io/os", func() {
			build := multiArchBuild(&MultiArch{
				Platforms: []ImagePlatform{
					{OS: "linux", Arch: "amd64"},
				},
			}, map[string]string{
				"kubernetes.io/os": "linux",
			})
			validate(build)
			Expect(*build.Status.Reason).To(Equal(MultiArchNodeSelectorConflict))
			Expect(*build.Status.Message).To(ContainSubstring("kubernetes.io/os"))
		})

		It("should fail when nodeSelector contains kubernetes.io/arch", func() {
			build := multiArchBuild(&MultiArch{
				Platforms: []ImagePlatform{
					{OS: "linux", Arch: "amd64"},
				},
			}, map[string]string{
				"kubernetes.io/arch": "amd64",
			})
			validate(build)
			Expect(*build.Status.Reason).To(Equal(MultiArchNodeSelectorConflict))
			Expect(*build.Status.Message).To(ContainSubstring("kubernetes.io/arch"))
		})

		It("should pass when nodeSelector has unrelated labels", func() {
			build := multiArchBuild(&MultiArch{
				Platforms: []ImagePlatform{
					{OS: "linux", Arch: "amd64"},
				},
			}, map[string]string{
				"disktype": "ssd",
			})
			validate(build)
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})
	})
})

var _ = Describe("ValidateMultiArchExecutor", func() {
	It("should pass when executor is PipelineRun", func() {
		valid, reason, msg := validate.ValidateMultiArchExecutor("PipelineRun")
		Expect(valid).To(BeTrue())
		Expect(reason).To(BeEmpty())
		Expect(msg).To(BeEmpty())
	})

	It("should fail when executor is TaskRun", func() {
		valid, reason, msg := validate.ValidateMultiArchExecutor("TaskRun")
		Expect(valid).To(BeFalse())
		Expect(reason).To(Equal(string(MultiArchExecutorNotPipelineRun)))
		Expect(msg).To(ContainSubstring("PipelineRun executor mode"))
		Expect(msg).To(ContainSubstring("TaskRun"))
	})
})

var _ = Describe("ValidateMultiArchNodes", func() {
	var (
		ctx        context.Context
		fakeClient *fakes.FakeClient
	)

	BeforeEach(func() {
		ctx = context.TODO()
		fakeClient = &fakes.FakeClient{}
	})

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
		fakeClient.ListStub = func(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
			nodeList := list.(*corev1.NodeList)
			nodeList.Items = []corev1.Node{
				readyNode("node-amd64", "linux", "amd64"),
				readyNode("node-arm64", "linux", "arm64"),
			}
			return nil
		}
		platforms := []ImagePlatform{
			{OS: "linux", Arch: "amd64"},
			{OS: "linux", Arch: "arm64"},
		}
		valid, reason, msg := validate.ValidateMultiArchNodes(ctx, fakeClient, platforms)
		Expect(valid).To(BeTrue())
		Expect(reason).To(BeEmpty())
		Expect(msg).To(BeEmpty())
	})

	It("should fail when no node exists for a platform", func() {
		fakeClient.ListStub = func(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
			nodeList := list.(*corev1.NodeList)
			nodeList.Items = []corev1.Node{
				readyNode("node-amd64", "linux", "amd64"),
			}
			return nil
		}
		platforms := []ImagePlatform{
			{OS: "linux", Arch: "amd64"},
			{OS: "linux", Arch: "s390x"},
		}
		valid, reason, msg := validate.ValidateMultiArchNodes(ctx, fakeClient, platforms)
		Expect(valid).To(BeFalse())
		Expect(reason).To(Equal(string(MultiArchNodeNotFound)))
		Expect(msg).To(ContainSubstring("linux/s390x"))
	})

	// tests availablePlatforms function
	It("should skip unschedulable nodes", func() {
		fakeClient.ListStub = func(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
			nodeList := list.(*corev1.NodeList)
			unschedulable := readyNode("node-amd64", "linux", "amd64")
			unschedulable.Spec.Unschedulable = true
			nodeList.Items = []corev1.Node{unschedulable}
			return nil
		}
		platforms := []ImagePlatform{{OS: "linux", Arch: "amd64"}}
		valid, _, msg := validate.ValidateMultiArchNodes(ctx, fakeClient, platforms)
		Expect(valid).To(BeFalse())
		Expect(msg).To(ContainSubstring("linux/amd64"))
	})

	It("should skip nodes that are not Ready", func() {
		fakeClient.ListStub = func(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
			nodeList := list.(*corev1.NodeList)
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
			nodeList.Items = []corev1.Node{notReady}
			return nil
		}
		platforms := []ImagePlatform{{OS: "linux", Arch: "amd64"}}
		valid, _, msg := validate.ValidateMultiArchNodes(ctx, fakeClient, platforms)
		Expect(valid).To(BeFalse())
		Expect(msg).To(ContainSubstring("linux/amd64"))
	})

	It("should fail when List returns an error", func() {
		fakeClient.ListStub = func(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
			return fmt.Errorf("connection refused")
		}
		platforms := []ImagePlatform{{OS: "linux", Arch: "amd64"}}
		valid, reason, msg := validate.ValidateMultiArchNodes(ctx, fakeClient, platforms)
		Expect(valid).To(BeFalse())
		Expect(reason).To(Equal(string(MultiArchNodeNotFound)))
		Expect(msg).To(ContainSubstring("failed to list nodes"))
	})
})


// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package conversion

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	buildapialpha "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

var _ = Describe("BuildStrategy Conversion", func() {
	ctx := context.TODO()

	Context("convertBuildStrategyBetaToAlpha", func() {
		It("converts a standard BuildStrategy from v1beta1 to v1alpha1", func() {
			user := int64(1000)
			betaStrategy := &buildapi.BuildStrategy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: betaGroupVersion,
					Kind:       buildStrategyKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "custom-strategy",
					Namespace: "default",
				},
				Spec: buildapi.BuildStrategySpec{
					Parameters: []buildapi.Parameter{
						{
							Name:        "custom-param",
							Description: "A custom parameter",
							Type:        buildapi.ParameterTypeString,
							Default:     ptr.To("default-val"),
						},
						{
							Name:        "array-param",
							Description: "Array parameter",
							Type:        buildapi.ParameterTypeArray,
							Defaults:    &[]string{"a", "b"},
						},
					},
					Steps: []buildapi.Step{
						{
							Name:       "step-1",
							Image:      "golang:1.22",
							Command:    []string{"go"},
							Args:       []string{"build", "-v"},
							WorkingDir: "/workspace",
							Env: []corev1.EnvVar{
								{Name: "CGO_ENABLED", Value: "0"},
							},
						},
					},
					SecurityContext: &buildapi.BuildStrategySecurityContext{
						RunAsUser: user,
					},
					Volumes: []buildapi.BuildStrategyVolume{
						{
							Name:        "cache-vol",
							Description: ptr.To("Cache storage"),
							Overridable: ptr.To(true),
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			err := convertBuildStrategyBetaToAlpha(ctx, betaStrategy, unstructuredObj)
			Expect(err).ToNot(HaveOccurred())

			var alphaStrategy buildapialpha.BuildStrategy
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), &alphaStrategy)
			Expect(err).ToNot(HaveOccurred())

			Expect(alphaStrategy.APIVersion).To(Equal(alphaGroupVersion))
			Expect(alphaStrategy.Name).To(Equal("custom-strategy"))

			// Parameters
			Expect(alphaStrategy.Spec.Parameters).To(HaveLen(2))
			Expect(alphaStrategy.Spec.Parameters[0].Name).To(Equal("custom-param"))
			Expect(alphaStrategy.Spec.Parameters[0].Default).To(Equal(ptr.To("default-val")))
			Expect(alphaStrategy.Spec.Parameters[1].Name).To(Equal("array-param"))
			Expect(*alphaStrategy.Spec.Parameters[1].Defaults).To(Equal([]string{"a", "b"}))

			// Steps
			Expect(alphaStrategy.Spec.BuildSteps).To(HaveLen(1))
			Expect(alphaStrategy.Spec.BuildSteps[0].Name).To(Equal("step-1"))
			Expect(alphaStrategy.Spec.BuildSteps[0].Image).To(Equal("golang:1.22"))
			Expect(alphaStrategy.Spec.BuildSteps[0].Command).To(Equal([]string{"go"}))
			Expect(alphaStrategy.Spec.BuildSteps[0].Args).To(Equal([]string{"build", "-v"}))

			// SecurityContext and Volumes
			Expect(alphaStrategy.Spec.SecurityContext).ToNot(BeNil())
			Expect(alphaStrategy.Spec.SecurityContext.RunAsUser).To(Equal(user))
			Expect(alphaStrategy.Spec.Volumes).To(HaveLen(1))
			Expect(alphaStrategy.Spec.Volumes[0].Name).To(Equal("cache-vol"))
			Expect(alphaStrategy.Spec.Volumes[0].Overridable).To(Equal(ptr.To(true)))
		})

		It("migrates dockerfile and builder-image parameters and references from Beta to Alpha", func() {
			betaStrategy := &buildapi.BuildStrategy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: betaGroupVersion,
					Kind:       buildStrategyKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "dockerfile-strategy",
				},
				Spec: buildapi.BuildStrategySpec{
					Parameters: []buildapi.Parameter{
						{
							Name:    "dockerfile",
							Type:    buildapi.ParameterTypeString,
							Default: ptr.To("Dockerfile"),
						},
						{
							Name: "builder-image",
							Type: buildapi.ParameterTypeString,
						},
						{
							Name:    "keep-me",
							Type:    buildapi.ParameterTypeString,
							Default: ptr.To("keep"),
						},
					},
					Steps: []buildapi.Step{
						{
							Name:    "build",
							Image:   "kaniko:latest",
							Command: []string{"kaniko", "--dockerfile=$(params.dockerfile)", "--builder=$(params.builder-image)"},
							Args:    []string{"--dockerfile=$(params.dockerfile)", "--builder=$(params.builder-image)"},
							Env: []corev1.EnvVar{
								{Name: "DOCKERFILE", Value: "$(params.dockerfile)"},
								{Name: "BUILDER", Value: "$(params.builder-image)"},
							},
						},
					},
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			err := convertBuildStrategyBetaToAlpha(ctx, betaStrategy, unstructuredObj)
			Expect(err).ToNot(HaveOccurred())

			var alphaStrategy buildapialpha.BuildStrategy
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), &alphaStrategy)
			Expect(err).ToNot(HaveOccurred())

			// Migrated parameters must be omitted from alpha parameters
			Expect(alphaStrategy.Spec.Parameters).To(HaveLen(1))
			Expect(alphaStrategy.Spec.Parameters[0].Name).To(Equal("keep-me"))

			// References must be migrated to legacy formats in Command, Args, and Env
			step := alphaStrategy.Spec.BuildSteps[0]
			Expect(step.Command[1]).To(Equal("--dockerfile=$(params.DOCKERFILE)"))
			Expect(step.Command[2]).To(Equal("--builder=$(build.builder.image)"))
			Expect(step.Args[0]).To(Equal("--dockerfile=$(params.DOCKERFILE)"))
			Expect(step.Args[1]).To(Equal("--builder=$(build.builder.image)"))
			Expect(step.Env[0].Value).To(Equal("$(params.DOCKERFILE)"))
			Expect(step.Env[1].Value).To(Equal("$(build.builder.image)"))
		})
	})

	Context("convertBuildStrategyAlphaToBeta", func() {
		It("migrates legacy DOCKERFILE and builder.image references from Alpha to Beta", func() {
			alphaStrategy := &buildapialpha.BuildStrategy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: alphaGroupVersion,
					Kind:       buildStrategyKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "legacy-alpha-strategy",
				},
				Spec: buildapialpha.BuildStrategySpec{
					BuildSteps: []buildapialpha.BuildStep{
						{
							Container: corev1.Container{
								Name:    "buildah",
								Image:   "buildah:latest",
								Command: []string{"buildah", "--dockerfile=$(build.dockerfile)", "--builder=$(build.builder.image)"},
								Args:    []string{"--df=$(params.DOCKERFILE)"},
								Env: []corev1.EnvVar{
									{Name: "DF1", Value: "$(params.DOCKERFILE)"},
									{Name: "DF2", Value: "$(build.dockerfile)"},
									{Name: "B1", Value: "$(build.builder.image)"},
								},
							},
						},
					},
					Parameters: []buildapialpha.Parameter{
						{
							Name:    "existing-param",
							Type:    buildapialpha.ParameterTypeString,
							Default: ptr.To("foo"),
						},
					},
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			mapito, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alphaStrategy)
			Expect(err).ToNot(HaveOccurred())
			unstructuredObj.SetUnstructuredContent(mapito)

			var betaStrategy buildapi.BuildStrategy
			err = convertBuildStrategyAlphaToBeta(ctx, &betaStrategy, unstructuredObj)
			Expect(err).ToNot(HaveOccurred())

			// References must be migrated
			step := betaStrategy.Spec.Steps[0]
			Expect(step.Command[1]).To(Equal("--dockerfile=$(params.dockerfile)"))
			Expect(step.Command[2]).To(Equal("--builder=$(params.builder-image)"))
			Expect(step.Args[0]).To(Equal("--df=$(params.dockerfile)"))
			Expect(step.Env[0].Value).To(Equal("$(params.dockerfile)"))
			Expect(step.Env[1].Value).To(Equal("$(params.dockerfile)"))
			Expect(step.Env[2].Value).To(Equal("$(params.builder-image)"))

			// Auto-added replacement parameters
			Expect(betaStrategy.Spec.Parameters).To(HaveLen(3))
			Expect(betaStrategy.Spec.Parameters[0].Name).To(Equal("existing-param"))
			Expect(betaStrategy.Spec.Parameters[1].Name).To(Equal("dockerfile"))
			Expect(betaStrategy.Spec.Parameters[1].Default).To(Equal(ptr.To("Dockerfile")))
			Expect(betaStrategy.Spec.Parameters[2].Name).To(Equal("builder-image"))
		})
	})

	Context("Round-trip BuildStrategy conversions", func() {
		It("preserves data in a round trip beta -> alpha -> beta", func() {
			originalBeta := &buildapi.BuildStrategy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: betaGroupVersion,
					Kind:       buildStrategyKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "rt-strategy",
				},
				Spec: buildapi.BuildStrategySpec{
					Parameters: []buildapi.Parameter{
						{
							Name:        "timeout",
							Type:        buildapi.ParameterTypeString,
							Default:     ptr.To("60s"),
							Description: "Build timeout",
						},
					},
					Steps: []buildapi.Step{
						{
							Name:    "compile",
							Image:   "golang:alpine",
							Command: []string{"go", "build"},
						},
					},
				},
			}

			// Beta -> Alpha
			alphaUnstructured := &unstructured.Unstructured{}
			err := convertBuildStrategyBetaToAlpha(ctx, originalBeta, alphaUnstructured)
			Expect(err).ToNot(HaveOccurred())

			// Alpha -> Beta
			var convertedBeta buildapi.BuildStrategy
			err = convertBuildStrategyAlphaToBeta(ctx, &convertedBeta, alphaUnstructured)
			Expect(err).ToNot(HaveOccurred())

			Expect(convertedBeta.Name).To(Equal(originalBeta.Name))
			Expect(convertedBeta.Spec.Parameters[0].Name).To(Equal("timeout"))
			Expect(convertedBeta.Spec.Parameters[0].Default).To(Equal(ptr.To("60s")))
			Expect(convertedBeta.Spec.Steps[0].Name).To(Equal("compile"))
			Expect(convertedBeta.Spec.Steps[0].Image).To(Equal("golang:alpine"))
		})
	})
})

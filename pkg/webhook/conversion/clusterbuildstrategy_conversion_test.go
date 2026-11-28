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

var _ = Describe("ClusterBuildStrategy Conversion", func() {
	ctx := context.TODO()

	Context("convertClusterBuildStrategyBetaToAlpha", func() {
		It("converts a ClusterBuildStrategy from v1beta1 to v1alpha1", func() {
			user := int64(0)
			betaCBS := &buildapi.ClusterBuildStrategy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: betaGroupVersion,
					Kind:       clusterBuildStrategyKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-buildah",
				},
				Spec: buildapi.BuildStrategySpec{
					Parameters: []buildapi.Parameter{
						{
							Name:        "storage-driver",
							Description: "The storage driver for buildah",
							Type:        buildapi.ParameterTypeString,
							Default:     ptr.To("vfs"),
						},
					},
					Steps: []buildapi.Step{
						{
							Name:       "build",
							Image:      "quay.io/buildah/stable:latest",
							Command:    []string{"buildah", "bud"},
							WorkingDir: "/workspace",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: &user,
							},
						},
					},
					Volumes: []buildapi.BuildStrategyVolume{
						{
							Name: "var-lib-containers",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			err := convertClusterBuildStrategyBetaToAlpha(ctx, betaCBS, unstructuredObj)
			Expect(err).ToNot(HaveOccurred())

			var alphaCBS buildapialpha.ClusterBuildStrategy
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), &alphaCBS)
			Expect(err).ToNot(HaveOccurred())

			Expect(alphaCBS.APIVersion).To(Equal(alphaGroupVersion))
			Expect(alphaCBS.Name).To(Equal("cluster-buildah"))
			Expect(alphaCBS.Spec.Parameters).To(HaveLen(1))
			Expect(alphaCBS.Spec.Parameters[0].Name).To(Equal("storage-driver"))
			Expect(alphaCBS.Spec.Parameters[0].Default).To(Equal(ptr.To("vfs")))
			Expect(alphaCBS.Spec.BuildSteps).To(HaveLen(1))
			Expect(alphaCBS.Spec.BuildSteps[0].Name).To(Equal("build"))
			Expect(alphaCBS.Spec.BuildSteps[0].Image).To(Equal("quay.io/buildah/stable:latest"))
			Expect(alphaCBS.Spec.Volumes).To(HaveLen(1))
			Expect(alphaCBS.Spec.Volumes[0].Name).To(Equal("var-lib-containers"))
		})
	})

	Context("convertClusterBuildStrategyAlphaToBeta", func() {
		It("converts a ClusterBuildStrategy from v1alpha1 to v1beta1", func() {
			alphaCBS := &buildapialpha.ClusterBuildStrategy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: alphaGroupVersion,
					Kind:       clusterBuildStrategyKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-kaniko",
				},
				Spec: buildapialpha.BuildStrategySpec{
					Parameters: []buildapialpha.Parameter{
						{
							Name:    "insecure",
							Type:    buildapialpha.ParameterTypeString,
							Default: ptr.To("false"),
						},
					},
					BuildSteps: []buildapialpha.BuildStep{
						{
							Container: corev1.Container{
								Name:    "executor",
								Image:   "gcr.io/kaniko-project/executor:latest",
								Command: []string{"/kaniko/executor"},
							},
						},
					},
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			mapito, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alphaCBS)
			Expect(err).ToNot(HaveOccurred())
			unstructuredObj.SetUnstructuredContent(mapito)

			var betaCBS buildapi.ClusterBuildStrategy
			err = convertClusterBuildStrategyAlphaToBeta(ctx, &betaCBS, unstructuredObj)
			Expect(err).ToNot(HaveOccurred())

			Expect(betaCBS.APIVersion).To(Equal(betaGroupVersion))
			Expect(betaCBS.Name).To(Equal("cluster-kaniko"))
			Expect(betaCBS.Spec.Parameters).To(HaveLen(1))
			Expect(betaCBS.Spec.Parameters[0].Name).To(Equal("insecure"))
			Expect(betaCBS.Spec.Parameters[0].Default).To(Equal(ptr.To("false")))
			Expect(betaCBS.Spec.Steps).To(HaveLen(1))
			Expect(betaCBS.Spec.Steps[0].Name).To(Equal("executor"))
		})
	})

	Context("Round-trip ClusterBuildStrategy conversions", func() {
		It("preserves data in a round trip beta -> alpha -> beta", func() {
			originalBeta := &buildapi.ClusterBuildStrategy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: betaGroupVersion,
					Kind:       clusterBuildStrategyKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "rt-cbs",
				},
				Spec: buildapi.BuildStrategySpec{
					Parameters: []buildapi.Parameter{
						{
							Name:    "arg",
							Type:    buildapi.ParameterTypeString,
							Default: ptr.To("val"),
						},
					},
					Steps: []buildapi.Step{
						{
							Name:  "step-1",
							Image: "alpine:latest",
						},
					},
				},
			}

			// Beta -> Alpha
			alphaUnstructured := &unstructured.Unstructured{}
			err := convertClusterBuildStrategyBetaToAlpha(ctx, originalBeta, alphaUnstructured)
			Expect(err).ToNot(HaveOccurred())

			// Alpha -> Beta
			var convertedBeta buildapi.ClusterBuildStrategy
			err = convertClusterBuildStrategyAlphaToBeta(ctx, &convertedBeta, alphaUnstructured)
			Expect(err).ToNot(HaveOccurred())

			Expect(convertedBeta.Name).To(Equal(originalBeta.Name))
			Expect(convertedBeta.Spec.Parameters[0].Name).To(Equal("arg"))
			Expect(convertedBeta.Spec.Parameters[0].Default).To(Equal(ptr.To("val")))
			Expect(convertedBeta.Spec.Steps[0].Name).To(Equal("step-1"))
		})
	})
})

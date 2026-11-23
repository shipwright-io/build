// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package conversion

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("convertSHPCR and status helpers", func() {
	ctx := context.TODO()

	Context("same version conversion", func() {
		It("returns identical object with success when fromVersion equals toVersion", func() {
			u := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": betaGroupVersion,
					"kind":       buildKind,
					"metadata": map[string]interface{}{
						"name": "noop-build",
					},
				},
			}

			res, status := convertSHPCR(ctx, u, betaGroupVersion)
			Expect(status.Status).To(Equal(metav1.StatusSuccess))
			Expect(res).ToNot(BeNil())
			Expect(res.GetName()).To(Equal("noop-build"))
		})
	})

	Context("unsupported fromVersion", func() {
		It("returns failure status when fromVersion is unexpected", func() {
			u := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "shipwright.io/v1gamma1",
					"kind":       buildKind,
				},
			}

			res, status := convertSHPCR(ctx, u, alphaGroupVersion)
			Expect(status.Status).To(Equal(metav1.StatusFailure))
			Expect(status.Message).To(ContainSubstring("unexpected conversion version from"))
			Expect(res).To(BeNil())
		})
	})

	Context("unsupported toVersion", func() {
		It("returns failure status when converting from beta to unknown toVersion", func() {
			u := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": betaGroupVersion,
					"kind":       buildKind,
				},
			}

			res, status := convertSHPCR(ctx, u, "shipwright.io/v2")
			Expect(status.Status).To(Equal(metav1.StatusFailure))
			Expect(status.Message).To(ContainSubstring("unexpected conversion version to"))
			Expect(res).To(BeNil())
		})

		It("returns failure status when converting from alpha to unknown toVersion", func() {
			u := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": alphaGroupVersion,
					"kind":       buildKind,
				},
			}

			res, status := convertSHPCR(ctx, u, "shipwright.io/v2")
			Expect(status.Status).To(Equal(metav1.StatusFailure))
			Expect(status.Message).To(ContainSubstring("unexpected conversion version to"))
			Expect(res).To(BeNil())
		})
	})

	Context("unsupported Kind", func() {
		It("returns failure status when converting an unknown Kind from beta to alpha", func() {
			u := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": betaGroupVersion,
					"kind":       "UnknownCustomResource",
				},
			}

			res, status := convertSHPCR(ctx, u, alphaGroupVersion)
			Expect(status.Status).To(Equal(metav1.StatusFailure))
			Expect(status.Message).To(Equal("unsupported Kind"))
			Expect(res).To(BeNil())
		})

		It("returns failure status when converting an unknown Kind from alpha to beta", func() {
			u := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": alphaGroupVersion,
					"kind":       "UnknownCustomResource",
				},
			}

			res, status := convertSHPCR(ctx, u, betaGroupVersion)
			Expect(status.Status).To(Equal(metav1.StatusFailure))
			Expect(status.Message).To(Equal("unsupported Kind"))
			Expect(res).To(BeNil())
		})
	})

	Context("All CRD conversions through convertSHPCR", func() {
		It("converts Build from beta to alpha and alpha to beta", func() {
			betaBuild := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": betaGroupVersion,
					"kind":       buildKind,
					"metadata": map[string]interface{}{
						"name": "shpcrd-build",
					},
					"spec": map[string]interface{}{
						"strategy": map[string]interface{}{
							"name": "buildah",
						},
					},
				},
			}

			alphaObj, status := convertSHPCR(ctx, betaBuild, alphaGroupVersion)
			Expect(status.Status).To(Equal(metav1.StatusSuccess))
			Expect(alphaObj.GetAPIVersion()).To(Equal(alphaGroupVersion))

			betaObj, status := convertSHPCR(ctx, alphaObj, betaGroupVersion)
			Expect(status.Status).To(Equal(metav1.StatusSuccess))
			Expect(betaObj.GetAPIVersion()).To(Equal(betaGroupVersion))
		})

		It("converts BuildRun from beta to alpha and alpha to beta", func() {
			betaBuildRun := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": betaGroupVersion,
					"kind":       buildRunKind,
					"metadata": map[string]interface{}{
						"name": "shpcrd-buildrun",
					},
					"spec": map[string]interface{}{
						"build": map[string]interface{}{
							"name": "target-build",
						},
					},
				},
			}

			alphaObj, status := convertSHPCR(ctx, betaBuildRun, alphaGroupVersion)
			Expect(status.Status).To(Equal(metav1.StatusSuccess))
			Expect(alphaObj.GetAPIVersion()).To(Equal(alphaGroupVersion))

			betaObj, status := convertSHPCR(ctx, alphaObj, betaGroupVersion)
			Expect(status.Status).To(Equal(metav1.StatusSuccess))
			Expect(betaObj.GetAPIVersion()).To(Equal(betaGroupVersion))
		})

		It("converts BuildStrategy from beta to alpha and alpha to beta", func() {
			betaBuildStrategy := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": betaGroupVersion,
					"kind":       buildStrategyKind,
					"metadata": map[string]interface{}{
						"name": "shpcrd-strategy",
					},
					"spec": map[string]interface{}{
						"steps": []interface{}{
							map[string]interface{}{
								"name":  "build",
								"image": "alpine:latest",
							},
						},
					},
				},
			}

			alphaObj, status := convertSHPCR(ctx, betaBuildStrategy, alphaGroupVersion)
			Expect(status.Status).To(Equal(metav1.StatusSuccess))
			Expect(alphaObj.GetAPIVersion()).To(Equal(alphaGroupVersion))

			betaObj, status := convertSHPCR(ctx, alphaObj, betaGroupVersion)
			Expect(status.Status).To(Equal(metav1.StatusSuccess))
			Expect(betaObj.GetAPIVersion()).To(Equal(betaGroupVersion))
		})

		It("converts ClusterBuildStrategy from beta to alpha and alpha to beta", func() {
			betaCBS := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": betaGroupVersion,
					"kind":       clusterBuildStrategyKind,
					"metadata": map[string]interface{}{
						"name": "cluster-strategy",
					},
					"spec": map[string]interface{}{
						"steps": []interface{}{
							map[string]interface{}{
								"name":  "build",
								"image": "busybox:latest",
							},
						},
					},
				},
			}

			alphaObj, status := convertSHPCR(ctx, betaCBS, alphaGroupVersion)
			Expect(status.Status).To(Equal(metav1.StatusSuccess))
			Expect(alphaObj.GetAPIVersion()).To(Equal(alphaGroupVersion))

			betaObj, status := convertSHPCR(ctx, alphaObj, betaGroupVersion)
			Expect(status.Status).To(Equal(metav1.StatusSuccess))
			Expect(betaObj.GetAPIVersion()).To(Equal(betaGroupVersion))
		})
	})

	Context("unstructuring error handling", func() {
		It("returns failure status when beta Build has invalid field types", func() {
			u := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": betaGroupVersion,
					"kind":       buildKind,
					"spec": map[string]interface{}{
						"source": "this-should-be-a-map-not-a-string",
					},
				},
			}

			res, status := convertSHPCR(ctx, u, alphaGroupVersion)
			Expect(status.Status).To(Equal(metav1.StatusFailure))
			Expect(status.Reason).To(Equal(metav1.StatusReasonInternalError))
			Expect(res).To(BeNil())
		})

		It("returns failure status when beta BuildRun has invalid field types", func() {
			u := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": betaGroupVersion,
					"kind":       buildRunKind,
					"spec": map[string]interface{}{
						"build": "invalid-build-type",
					},
				},
			}

			res, status := convertSHPCR(ctx, u, alphaGroupVersion)
			Expect(status.Status).To(Equal(metav1.StatusFailure))
			Expect(status.Reason).To(Equal(metav1.StatusReasonInternalError))
			Expect(res).To(BeNil())
		})

		It("returns failure status when beta BuildStrategy has invalid field types", func() {
			u := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": betaGroupVersion,
					"kind":       buildStrategyKind,
					"spec": map[string]interface{}{
						"steps": "invalid-steps-type",
					},
				},
			}

			res, status := convertSHPCR(ctx, u, alphaGroupVersion)
			Expect(status.Status).To(Equal(metav1.StatusFailure))
			Expect(status.Reason).To(Equal(metav1.StatusReasonInternalError))
			Expect(res).To(BeNil())
		})

		It("returns failure status when beta ClusterBuildStrategy has invalid field types", func() {
			u := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": betaGroupVersion,
					"kind":       clusterBuildStrategyKind,
					"spec": map[string]interface{}{
						"steps": "invalid-steps-type",
					},
				},
			}

			res, status := convertSHPCR(ctx, u, alphaGroupVersion)
			Expect(status.Status).To(Equal(metav1.StatusFailure))
			Expect(status.Reason).To(Equal(metav1.StatusReasonInternalError))
			Expect(res).To(BeNil())
		})
	})
})

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	crc "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/shipwright-io/build/pkg/controller/fakes"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
)

var _ = Describe("CheckVolumesExist", func() {

	var (
		client    *fakes.FakeClient
		namespace string
	)

	BeforeEach(func() {
		namespace = "test-ns"
		client = &fakes.FakeClient{}
	})

	Context("for a namepace where my-configmap and my-secret exist", func() {

		BeforeEach(func() {
			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, _ ...crc.GetOption) error {
				if nn.Name == "my-secret" || nn.Name == "my-configmap" {
					return nil
				}
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			})
		})

		It("succeeds for volumes that only reference those objects", func(ctx SpecContext) {
			vols := []corev1.Volume{{
				Name: "sec-vol",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "my-secret",
					},
				},
			}, {
				Name: "cm-vol",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "my-configmap",
						},
					},
				},
			}}

			Expect(resources.CheckVolumesExist(ctx, client, namespace, vols)).To(Succeed())
		})

		It("fails for volumes that reference something else", func(ctx SpecContext) {
			vols := []corev1.Volume{{
				Name: "sec-vol",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "missing-secret",
					},
				},
			}}

			err := resources.CheckVolumesExist(ctx, client, namespace, vols)
			Expect(err).To(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("succeeds for projected volumes that only reference those objects", func(ctx SpecContext) {
			vols := []corev1.Volume{{
				Name: "proj-vol",
				VolumeSource: corev1.VolumeSource{
					Projected: &corev1.ProjectedVolumeSource{
						Sources: []corev1.VolumeProjection{
							{
								Secret: &corev1.SecretProjection{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "my-secret",
									},
								},
							},
							{
								ConfigMap: &corev1.ConfigMapProjection{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "my-configmap",
									},
								},
							},
						},
					},
				},
			}}

			Expect(resources.CheckVolumesExist(ctx, client, namespace, vols)).To(Succeed())
		})

		It("succeeds for projected volumes with sources that are neither secret nor config map", func(ctx SpecContext) {
			vols := []corev1.Volume{{
				Name: "proj-vol",
				VolumeSource: corev1.VolumeSource{
					Projected: &corev1.ProjectedVolumeSource{
						Sources: []corev1.VolumeProjection{
							{
								ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
									Path: "token",
								},
							},
						},
					},
				},
			}}

			Expect(resources.CheckVolumesExist(ctx, client, namespace, vols)).To(Succeed())
		})

		It("fails for projected volumes referencing a missing secret", func(ctx SpecContext) {
			vols := []corev1.Volume{{
				Name: "proj-vol",
				VolumeSource: corev1.VolumeSource{
					Projected: &corev1.ProjectedVolumeSource{
						Sources: []corev1.VolumeProjection{
							{
								ConfigMap: &corev1.ConfigMapProjection{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "my-configmap",
									},
								},
							},
							{
								Secret: &corev1.SecretProjection{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "missing-secret",
									},
								},
							},
						},
					},
				},
			}}

			err := resources.CheckVolumesExist(ctx, client, namespace, vols)
			Expect(err).To(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("fails for projected volumes referencing a missing config map", func(ctx SpecContext) {
			vols := []corev1.Volume{{
				Name: "proj-vol",
				VolumeSource: corev1.VolumeSource{
					Projected: &corev1.ProjectedVolumeSource{
						Sources: []corev1.VolumeProjection{
							{
								ConfigMap: &corev1.ConfigMapProjection{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "missing-configmap",
									},
								},
							},
						},
					},
				},
			}}

			err := resources.CheckVolumesExist(ctx, client, namespace, vols)
			Expect(err).To(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})
	})
})

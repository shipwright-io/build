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

var _ = Describe("Volumes", func() {
	var (
		ctx       context.Context
		client    *fakes.FakeClient
		namespace string
	)

	BeforeEach(func() {
		ctx = context.TODO()
		namespace = "test-ns"
		client = &fakes.FakeClient{}
	})

	Context("when checking volumes exist", func() {
		It("succeeds when referenced Secret and ConfigMap exist", func() {
			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, _ ...crc.GetOption) error {
				if nn.Name == "my-secret" || nn.Name == "my-configmap" {
					return nil
				}
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			})

			vols := []corev1.Volume{
				{
					Name: "sec-vol",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "my-secret",
						},
					},
				},
				{
					Name: "cm-vol",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "my-configmap",
							},
						},
					},
				},
			}

			err := resources.CheckVolumesExist(ctx, client, namespace, vols)
			Expect(err).ToNot(HaveOccurred())
		})

		It("fails when referenced Secret is missing", func() {
			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, _ ...crc.GetOption) error {
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			})

			vols := []corev1.Volume{
				{
					Name: "sec-vol",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "missing-secret",
						},
					},
				},
			}

			err := resources.CheckVolumesExist(ctx, client, namespace, vols)
			Expect(err).To(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})
	})
})

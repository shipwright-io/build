// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
)

var _ = Describe("Credentials", func() {

	var (
		build                       *buildv1alpha1.Build
		beforeServiceAccount        *corev1.ServiceAccount
		expectedAfterServiceAccount *corev1.ServiceAccount
	)

	BeforeEach(func() {
		beforeServiceAccount = &corev1.ServiceAccount{
			Secrets: []corev1.ObjectReference{
				{Name: "secret_b"}, {Name: "secret_c"},
			},
		}
	})

	Context("when secrets were not present in the service account", func() {

		BeforeEach(func() {
			build = &buildv1alpha1.Build{
				Spec: buildv1alpha1.BuildSpec{
					Source: buildv1alpha1.Source{
						URL: "a/b/c",
						Credentials: &corev1.LocalObjectReference{
							Name: "secret_a",
						},
					},
					Builder: &buildv1alpha1.Image{
						Image: "quay.io/namespace/image",
						Credentials: &corev1.LocalObjectReference{
							Name: "secret_docker.io",
						},
					},
					Output: buildv1alpha1.Image{
						Image: "quay.io/namespace/image",
						Credentials: &corev1.LocalObjectReference{
							Name: "secret_quay.io",
						},
					},
				},
			}

			// source credential is not added to the service account
			expectedAfterServiceAccount = &corev1.ServiceAccount{
				Secrets: []corev1.ObjectReference{
					{Name: "secret_b"}, {Name: "secret_c"}, {Name: "secret_docker.io"}, {Name: "secret_quay.io"},
				},
			}
		})

		It("adds the credentials to the service account", func() {
			afterServiceAccount := beforeServiceAccount.DeepCopy()
			modified := resources.ApplyCredentials(context.TODO(), build, afterServiceAccount)

			Expect(modified).To(BeTrue())
			Expect(afterServiceAccount).To(Equal(expectedAfterServiceAccount))
		})
	})

	Context("when secrets were already in the service account", func() {

		BeforeEach(func() {
			build = &buildv1alpha1.Build{
				Spec: buildv1alpha1.BuildSpec{
					Source: buildv1alpha1.Source{
						URL: "a/b/c",
						Credentials: &corev1.LocalObjectReference{
							Name: "secret_b",
						},
					},
				},
			}

			expectedAfterServiceAccount = beforeServiceAccount
		})

		It("keeps the service account unchanged", func() {
			afterServiceAccount := beforeServiceAccount.DeepCopy()
			modified := resources.ApplyCredentials(context.TODO(), build, afterServiceAccount)

			Expect(modified).To(BeFalse())
			Expect(afterServiceAccount).To(Equal(expectedAfterServiceAccount))
		})
	})

	Context("when build does not reference any secret", func() {

		BeforeEach(func() {
			build = &buildv1alpha1.Build{
				Spec: buildv1alpha1.BuildSpec{
					Source: buildv1alpha1.Source{
						URL:         "a/b/c",
						Credentials: nil,
					},
				},
			}

			expectedAfterServiceAccount = beforeServiceAccount
		})

		It("keeps the service account unchanged", func() {
			afterServiceAccount := beforeServiceAccount.DeepCopy()
			modified := resources.ApplyCredentials(context.TODO(), build, afterServiceAccount)

			Expect(modified).To(BeFalse())
			Expect(afterServiceAccount).To(Equal(expectedAfterServiceAccount))
		})
	})
})

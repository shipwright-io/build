// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
)

var _ = Describe("Credentials", func() {

	var (
		build                       *buildv1alpha1.Build
		buildRun                    *buildv1alpha1.BuildRun
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
						URL: pointer.StringPtr("a/b/c"),
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

			buildRun = &buildv1alpha1.BuildRun{
				Spec: buildv1alpha1.BuildRunSpec{
					Output: &buildv1alpha1.Image{
						Image: "quay.io/namespace/brImage",
						Credentials: &corev1.LocalObjectReference{
							Name: "secret_buildrun.io",
						},
					},
				},
			}

			expectedAfterServiceAccount = &corev1.ServiceAccount{
				// source credential is not added to the service account
				Secrets: []corev1.ObjectReference{
					{Name: "secret_b"},
					{Name: "secret_c"},
					{Name: "secret_docker.io"},
					{Name: "secret_buildrun.io"},
				},
			}
		})

		It("adds the credentials to the service account", func() {
			afterServiceAccount := beforeServiceAccount.DeepCopy()
			modified := resources.ApplyCredentials(context.TODO(), build, buildRun, afterServiceAccount)

			Expect(modified).To(BeTrue())
			Expect(afterServiceAccount).To(Equal(expectedAfterServiceAccount))
		})
	})

	Context("when secrets were already in the service account", func() {

		BeforeEach(func() {
			build = &buildv1alpha1.Build{
				Spec: buildv1alpha1.BuildSpec{
					Source: buildv1alpha1.Source{
						URL: pointer.StringPtr("a/b/c"),
					},
					Output: buildv1alpha1.Image{
						Credentials: &corev1.LocalObjectReference{
							Name: "secret_b",
						},
					},
				},
			}

			// This is just a placeholder BuildRun with no
			// SecretRef added to the ones from the Build
			buildRun = &buildv1alpha1.BuildRun{
				Spec: buildv1alpha1.BuildRunSpec{
					Output: &buildv1alpha1.Image{
						Image: "https://image.url/",
					},
				},
			}

			expectedAfterServiceAccount = beforeServiceAccount
		})

		It("keeps the service account unchanged", func() {
			afterServiceAccount := beforeServiceAccount.DeepCopy()
			modified := resources.ApplyCredentials(context.TODO(), build, buildRun, afterServiceAccount)

			Expect(modified).To(BeFalse())
			Expect(afterServiceAccount).To(Equal(expectedAfterServiceAccount))
		})
	})

	Context("when build does not reference any secret", func() {

		BeforeEach(func() {
			build = &buildv1alpha1.Build{
				Spec: buildv1alpha1.BuildSpec{
					Source: buildv1alpha1.Source{
						URL:         pointer.StringPtr("a/b/c"),
						Credentials: nil,
					},
				},
			}

			// This is just a placeholder BuildRun with no
			// SecretRef added to the ones from the Build
			buildRun = &buildv1alpha1.BuildRun{
				Spec: buildv1alpha1.BuildRunSpec{
					Output: &buildv1alpha1.Image{
						Image: "https://image.url/",
					},
				},
			}

			expectedAfterServiceAccount = beforeServiceAccount
		})

		It("keeps the service account unchanged", func() {
			afterServiceAccount := beforeServiceAccount.DeepCopy()
			modified := resources.ApplyCredentials(context.TODO(), build, buildRun, afterServiceAccount)

			Expect(modified).To(BeFalse())
			Expect(afterServiceAccount).To(Equal(expectedAfterServiceAccount))
		})
	})
})

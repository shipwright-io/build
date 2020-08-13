package buildrun

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	buildRunController "github.com/redhat-developer/build/pkg/controller/buildrun"
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
					Source: buildv1alpha1.GitSource{
						URL: "a/b/c",
						SecretRef: &corev1.LocalObjectReference{
							Name: "secret_a",
						},
					},
					BuilderImage: &buildv1alpha1.Image{
						ImageURL: "quay.io/namespace/image",
						SecretRef: &corev1.LocalObjectReference{
							Name: "secret_docker.io",
						},
					},
					Output: buildv1alpha1.Image{
						ImageURL: "quay.io/namespace/image",
						SecretRef: &corev1.LocalObjectReference{
							Name: "secret_quay.io",
						},
					},
				},
			}

			expectedAfterServiceAccount = &corev1.ServiceAccount{
				Secrets: []corev1.ObjectReference{
					{Name: "secret_b"}, {Name: "secret_c"}, {Name: "secret_a"}, {Name: "secret_docker.io"}, {Name: "secret_quay.io"},
				},
			}
		})

		It("adds the credentials to the service account", func() {
			afterServiceAccount := beforeServiceAccount.DeepCopy()
			modified := buildRunController.ApplyCredentials(context.TODO(), build, afterServiceAccount)

			Expect(modified).To(BeTrue())
			Expect(afterServiceAccount).To(Equal(expectedAfterServiceAccount))
		})
	})

	Context("when secrets were already in the service account", func() {

		BeforeEach(func() {
			build = &buildv1alpha1.Build{
				Spec: buildv1alpha1.BuildSpec{
					Source: buildv1alpha1.GitSource{
						URL: "a/b/c",
						SecretRef: &corev1.LocalObjectReference{
							Name: "secret_b",
						},
					},
				},
			}

			expectedAfterServiceAccount = beforeServiceAccount
		})

		It("keeps the service account unchanged", func() {
			afterServiceAccount := beforeServiceAccount.DeepCopy()
			modified := buildRunController.ApplyCredentials(context.TODO(), build, afterServiceAccount)

			Expect(modified).To(BeFalse())
			Expect(afterServiceAccount).To(Equal(expectedAfterServiceAccount))
		})
	})

	Context("when build does not reference any secret", func() {

		BeforeEach(func() {
			build = &buildv1alpha1.Build{
				Spec: buildv1alpha1.BuildSpec{
					Source: buildv1alpha1.GitSource{
						URL:       "a/b/c",
						SecretRef: nil,
					},
				},
			}

			expectedAfterServiceAccount = beforeServiceAccount
		})

		It("keeps the service account unchanged", func() {
			afterServiceAccount := beforeServiceAccount.DeepCopy()
			modified := buildRunController.ApplyCredentials(context.TODO(), build, afterServiceAccount)

			Expect(modified).To(BeFalse())
			Expect(afterServiceAccount).To(Equal(expectedAfterServiceAccount))
		})
	})
})

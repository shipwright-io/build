// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/validate"
)

var _ = Describe("Env", func() {
	Context("ValidatePath", func() {
		It("should fail in case of empty env var name", func() {
			b := &build.Build{
				Spec: build.BuildSpec{
					Env: []corev1.EnvVar{
						{
							Name:  "",
							Value: "some-value",
						},
					},
				},
			}

			err := validate.NewEnv(b).ValidatePath(context.TODO())
			Expect(err).To(HaveOccurred())
			Expect(b.Status.Reason).To(Equal(build.BuildReasonPtr(build.SpecEnvNameCanNotBeBlank)))
			Expect(b.Status.Message).To(Equal(pointer.StringPtr("name for environment variable must not be blank")))
		})

		It("should fail in case of specifying both value and valueFrom", func() {
			b := &build.Build{
				Spec: build.BuildSpec{
					Env: []corev1.EnvVar{
						{
							Name:  "some-name",
							Value: "some-value",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "my-field-path",
								},
							},
						},
					},
				},
			}

			err := validate.NewEnv(b).ValidatePath(context.TODO())
			Expect(err).To(HaveOccurred())
			Expect(b.Status.Reason).To(Equal(build.BuildReasonPtr(build.SpecEnvOnlyOneOfValueOrValueFromMustBeSpecified)))
			Expect(b.Status.Message).To(Equal(pointer.StringPtr("only one of value or valueFrom must be specified")))
		})

		It("should pass in case no env var are set", func() {
			b := &build.Build{
				Spec: build.BuildSpec{},
			}

			err := validate.NewEnv(b).ValidatePath(context.TODO())
			Expect(err).To(BeNil())
		})

		It("should pass in case of compliant env var", func() {
			b := &build.Build{
				Spec: build.BuildSpec{
					Env: []corev1.EnvVar{
						{
							Name:  "some-name",
							Value: "some-value",
						},
					},
				},
			}

			err := validate.NewEnv(b).ValidatePath(context.TODO())
			Expect(err).To(BeNil())
		})

		It("should pass in case of compliant env var using valueFrom", func() {
			b := &build.Build{
				Spec: build.BuildSpec{
					Env: []corev1.EnvVar{
						{
							Name: "some-name",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "my-field-path",
								},
							},
						},
					},
				},
			}

			err := validate.NewEnv(b).ValidatePath(context.TODO())
			Expect(err).To(BeNil())
		})
	})
})

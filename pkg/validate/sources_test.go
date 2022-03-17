// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/validate"
)

var _ = Describe("SourcesRef", func() {
	Context("ValidatePath", func() {
		It("should successfully validate an empty sources slice", func() {
			srcRef := validate.NewSourcesRef(&build.Build{})

			Expect(srcRef.ValidatePath(context.TODO())).To(BeNil())
		})

		It("should successfully validate a build with a valid URL", func() {
			srcRef := validate.NewSourcesRef(&build.Build{
				Spec: build.BuildSpec{
					Sources: []build.BuildSource{
						{Name: "name", URL: "https://github.com/shipwright-io/build"},
					},
				},
			})

			Expect(srcRef.ValidatePath(context.TODO())).To(BeNil())
		})

		It("should fail to validate if the name is not informed", func() {
			srcRef := validate.NewSourcesRef(&build.Build{
				Spec: build.BuildSpec{
					Sources: []build.BuildSource{
						{Name: ""},
					},
				},
			})

			Expect(srcRef.ValidatePath(context.TODO())).To(HaveOccurred())
		})

		It("should fail to validate if the URL is not informed", func() {
			srcRef := validate.NewSourcesRef(&build.Build{
				Spec: build.BuildSpec{
					Sources: []build.BuildSource{
						{Name: "name", URL: ""},
					},
				},
			})

			Expect(srcRef.ValidatePath(context.TODO())).To(HaveOccurred())
		})

		It("should fail to validate a build with an invalid URL", func() {
			srcRef := validate.NewSourcesRef(&build.Build{
				Spec: build.BuildSpec{
					Sources: []build.BuildSource{
						{Name: "name", URL: "invalid URL"},
					},
				},
			})

			Expect(srcRef.ValidatePath(context.TODO())).To(HaveOccurred())
		})
	})
})

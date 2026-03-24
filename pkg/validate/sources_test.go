// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/validate"
)

var _ = Describe("SourcesRef", func() {
	Context("ValidatePath", func() {
		It("should successfully validate an empty source", func() {
			srcRef := validate.NewSourceRef(&buildapi.Build{})

			Expect(srcRef.ValidatePath(context.TODO())).To(BeNil())
		})

		It("should successfully validate a build with source", func() {
			srcRef := validate.NewSourceRef(&buildapi.Build{
				Spec: buildapi.BuildSpec{
					Source: &buildapi.Source{
						Type: buildapi.GitType,
						Git:  &buildapi.Git{},
					},
				},
			})

			Expect(srcRef.ValidatePath(context.TODO())).To(BeNil())
		})

		It("should fail to validate if the type is not defined", func() {
			srcRef := validate.NewSourceRef(&buildapi.Build{
				Spec: buildapi.BuildSpec{
					Source: &buildapi.Source{
						Git: &buildapi.Git{},
					},
				},
			})

			Expect(srcRef.ValidatePath(context.TODO())).To(HaveOccurred())
		})

		It("should fail to validate if the type does not match the source git", func() {
			srcRef := validate.NewSourceRef(&buildapi.Build{
				Spec: buildapi.BuildSpec{
					Source: &buildapi.Source{
						Type: buildapi.OCIArtifactType,
						Git:  &buildapi.Git{},
					},
				},
			})

			Expect(srcRef.ValidatePath(context.TODO())).To(HaveOccurred())
		})
	})
})

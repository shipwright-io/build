// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	buildapialpha "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

var _ = Describe("BuildSpec ConvertTo", func() {

	// verifies that converting a BuildSpec whose source type is OCIArtifact but
	// whose OCIArtifact field is unset does not panic, and instead produces an
	// empty BundleContainer, matching the existing nil-safe handling of Git sources
	It("does not panic when the source type is OCIArtifact but OCIArtifact is nil", func() {
		src := buildapi.BuildSpec{
			Source: &buildapi.Source{
				Type: buildapi.OCIArtifactType,
			},
		}

		dest := &buildapialpha.BuildSpec{}

		Expect(func() {
			Expect(src.ConvertTo(dest)).To(Succeed())
		}).ToNot(Panic())

		Expect(dest.Source.BundleContainer).To(BeNil())
	})
})

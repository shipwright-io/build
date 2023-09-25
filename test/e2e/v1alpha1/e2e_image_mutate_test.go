// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

var _ = Describe("For a Kubernetes cluster with Tekton and build installed", func() {
	var (
		err      error
		testID   string
		build    *buildv1alpha1.Build
		buildRun *buildv1alpha1.BuildRun
	)

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			printTestFailureDebugInfo(testBuild, testBuild.Namespace, testID)
		} else if buildRun != nil {
			validateServiceAccountDeletion(buildRun, testBuild.Namespace)
		}

		if buildRun != nil {
			testBuild.DeleteBR(buildRun.Name)
			buildRun = nil
		}

		if build != nil {
			testBuild.DeleteBuild(build.Name)
			build = nil
		}
	})

	Context("when a Buildah build with label and annotation is defined", func() {
		BeforeEach(func() {
			testID = generateTestID("buildah-mutate")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1alpha1/build_buildah_cr_mutate.yaml",
			)
		})

		It("should mutate an image with annotation and label", func() {
			buildRun, err = buildRunTestData(
				testBuild.Namespace, testID,
				"test/data/v1alpha1/buildrun_buildah_cr_mutate.yaml",
			)
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")
			appendRegistryInsecureParamValue(build, buildRun)

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)

			image := testBuild.GetImage(buildRun)

			Expect(
				getImageAnnotation(image, "org.opencontainers.image.url"),
			).To(Equal("https://my-company.com/images"))

			Expect(
				getImageLabel(image, "maintainer"),
			).To(Equal("team@my-company.com"))
		})
	})
})

func getImageAnnotation(img containerreg.Image, annotation string) string {
	manifest, err := img.Manifest()
	Expect(err).To(BeNil())

	return manifest.Annotations[annotation]
}

func getImageLabel(img containerreg.Image, label string) string {
	config, err := img.ConfigFile()
	Expect(err).To(BeNil())

	return config.Config.Labels[label]
}

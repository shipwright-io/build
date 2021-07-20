// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

var _ = Describe("For a Kubernetes cluster with Tekton and build installed", func() {
	var (
		testID string
		err    error

		build    *buildv1alpha1.Build
		buildRun *buildv1alpha1.BuildRun
	)

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
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

	Context("when using local source code bundle images as input", func() {
		var inputImage, outputImage string

		BeforeEach(func() {
			testID = generateTestID("bundle")

			inputImage = "quay.io/shipwright/source-bundle:latest"
			outputImage = fmt.Sprintf("%s/%s:%s",
				os.Getenv(EnvVarImageRepo),
				testID,
				"latest",
			)
		})

		It("should work with Kaniko build strategy", func() {
			build, err = NewBuildPrototype().
				ClusterBuildStrategy("kaniko").
				Name(testID).
				Namespace(testBuild.Namespace).
				SourceBundle(inputImage).
				SourceContextDir("docker-build").
				Dockerfile("Dockerfile").
				OutputImage(outputImage).
				Create()
			Expect(err).ToNot(HaveOccurred())

			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				ForBuild(build).
				GenerateServiceAccount().
				Create()
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(testBuild, buildRun)
		})

		It("should work with Buildpacks build strategy", func() {
			build, err = NewBuildPrototype().
				ClusterBuildStrategy("buildpacks-v3").
				Name(testID).
				Namespace(testBuild.Namespace).
				SourceBundle(inputImage).
				SourceContextDir("source-build").
				OutputImage(outputImage).
				Create()
			Expect(err).ToNot(HaveOccurred())

			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				ForBuild(build).
				GenerateServiceAccount().
				Create()
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(testBuild, buildRun)
		})

		It("should work with Buildah build strategy", func() {
			build, err = NewBuildPrototype().
				ClusterBuildStrategy("buildah").
				Name(testID).
				Namespace(testBuild.Namespace).
				SourceBundle(inputImage).
				SourceContextDir("docker-build").
				Dockerfile("Dockerfile").
				OutputImage(outputImage).
				Create()
			Expect(err).ToNot(HaveOccurred())

			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				ForBuild(build).
				GenerateServiceAccount().
				Create()
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(testBuild, buildRun)
		})
	})
})

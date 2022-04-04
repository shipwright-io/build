// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/go-containerregistry/pkg/name"
	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

var _ = Describe("Using One-Off Builds", func() {
	var (
		testID string
		err    error

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
	})

	Context("Embed BuildSpec in BuildRun", func() {
		var outputImage name.Reference

		BeforeEach(func() {
			testID = generateTestID("onoff")

			outputImage, err = name.ParseReference(fmt.Sprintf("%s/%s:%s",
				os.Getenv(EnvVarImageRepo),
				testID,
				"latest",
			))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should build an image using Buildpacks and a Git source", func() {
			buildRun, err = NewBuildRunPrototype().
				Namespace(testBuild.Namespace).
				Name(testID).
				WithBuildSpec(NewBuildPrototype().
					ClusterBuildStrategy("buildpacks-v3").
					Namespace(testBuild.Namespace).
					Name(testID).
					SourceGit("https://github.com/shipwright-io/sample-go.git").
					SourceContextDir("source-build").
					OutputImage(outputImage.String()).
					OutputImageCredentials(os.Getenv(EnvVarImageRepoSecret)).
					BuildSpec()).
				Create()
			Expect(err).ToNot(HaveOccurred())
			validateBuildRunToSucceed(testBuild, buildRun)
		})

		It("should build an image using Buildah and a Git source", func() {
			buildRun, err = NewBuildRunPrototype().
				Namespace(testBuild.Namespace).
				Name(testID).
				WithBuildSpec(NewBuildPrototype().
					ClusterBuildStrategy("buildah").
					Namespace(testBuild.Namespace).
					Name(testID).
					SourceGit("https://github.com/shipwright-io/sample-go.git").
					SourceContextDir("docker-build").
					Dockerfile("Dockerfile").
					ArrayParamValue("registries-insecure", outputImage.Context().RegistryStr()).
					OutputImage(outputImage.String()).
					OutputImageCredentials(os.Getenv(EnvVarImageRepoSecret)).
					BuildSpec()).
				Create()
			Expect(err).ToNot(HaveOccurred())
			validateBuildRunToSucceed(testBuild, buildRun)
		})

		It("should build an image using Buildpacks and a bundle source", func() {
			buildRun, err = NewBuildRunPrototype().
				Namespace(testBuild.Namespace).
				Name(testID).
				WithBuildSpec(NewBuildPrototype().
					ClusterBuildStrategy("buildpacks-v3").
					Namespace(testBuild.Namespace).
					Name(testID).
					SourceBundle("ghcr.io/shipwright-io/sample-go/source-bundle:latest").
					SourceContextDir("source-build").
					OutputImage(outputImage.String()).
					OutputImageCredentials(os.Getenv(EnvVarImageRepoSecret)).
					BuildSpec()).
				Create()
			Expect(err).ToNot(HaveOccurred())
			validateBuildRunToSucceed(testBuild, buildRun)
		})
	})
})

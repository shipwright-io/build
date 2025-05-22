// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"fmt"
	"os"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/go-containerregistry/pkg/name"
	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

var _ = Describe("Using One-Off Builds", Label("FEATURE:OneOffBuild", "CORE"),  func() {

	insecure := false
	value, found := os.LookupEnv(EnvVarImageRepoInsecure)
	if found {
		var err error
		insecure, err = strconv.ParseBool(value)
		Expect(err).ToNot(HaveOccurred())
	}

	var (
		testID string
		err    error

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

		It("should build an image using Buildpacks and a Git source", Label("FEATURE:Buildpacks", "FEATURE:GitSource"), func() {
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
					OutputImageInsecure(insecure).
					BuildSpec()).
				Create()
			Expect(err).ToNot(HaveOccurred())
			validateBuildRunToSucceed(testBuild, buildRun)
		})

		It("should build an image using Buildah and a Git source", Label("FEATURE:Buildah", "FEATURE:GitSource"),func() {
			buildRun, err = NewBuildRunPrototype().
				Namespace(testBuild.Namespace).
				Name(testID).
				WithBuildSpec(NewBuildPrototype().
					ClusterBuildStrategy("buildah-shipwright-managed-push").
					Namespace(testBuild.Namespace).
					Name(testID).
					SourceGit("https://github.com/shipwright-io/sample-go.git").
					SourceContextDir("docker-build").
					Dockerfile("Dockerfile").
					ArrayParamValue("registries-insecure", outputImage.Context().RegistryStr()).
					OutputImage(outputImage.String()).
					OutputImageCredentials(os.Getenv(EnvVarImageRepoSecret)).
					OutputImageInsecure(insecure).
					BuildSpec()).
				Create()
			Expect(err).ToNot(HaveOccurred())
			validateBuildRunToSucceed(testBuild, buildRun)
		})

		It("should build an image using Buildpacks and a bundle source", Label("FEATURE:Buildpacks", "FEATURE:BundleSource"),func() {
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
					OutputImageInsecure(insecure).
					BuildSpec()).
				Create()
			Expect(err).ToNot(HaveOccurred())
			validateBuildRunToSucceed(testBuild, buildRun)
		})
	})
})

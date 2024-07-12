// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"fmt"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/go-containerregistry/pkg/name"
	containerreg "github.com/google/go-containerregistry/pkg/v1"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

var _ = Describe("For a Kubernetes cluster with Tekton and build installed", func() {
	var (
		err      error
		testID   string
		build    *buildv1beta1.Build
		buildRun *buildv1beta1.BuildRun
	)

	annotationsOf := func(img containerreg.Image) map[string]string {
		GinkgoHelper()
		manifest, err := img.Manifest()
		Expect(err).To(BeNil())
		return manifest.Annotations
	}

	labelsOf := func(img containerreg.Image) map[string]string {
		GinkgoHelper()
		config, err := img.ConfigFile()
		Expect(err).To(BeNil())
		return config.Config.Labels
	}

	creationTimeOf := func(img containerreg.Image) time.Time {
		GinkgoHelper()
		cfg, err := img.ConfigFile()
		Expect(err).ToNot(HaveOccurred())
		return cfg.Created.Time
	}

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
				"test/data/v1beta1/build_buildah_cr_mutate.yaml",
			)
		})

		It("should mutate an image with annotation and label", func() {
			buildRun, err = buildRunTestData(
				testBuild.Namespace, testID,
				"test/data/v1beta1/buildrun_buildah_cr_mutate.yaml",
			)
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")
			appendRegistryInsecureParamValue(build, buildRun)

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)

			image := testBuild.GetImage(buildRun)
			Expect(annotationsOf(image)).To(HaveKeyWithValue("org.opencontainers.image.url", "https://my-company.com/images"))
			Expect(labelsOf(image)).To(HaveKeyWithValue("maintainer", "team@my-company.com"))
		})
	})

	Context("mutate image timestamp", func() {
		var outputImage name.Reference

		var insecure = func() bool {
			if val, ok := os.LookupEnv(EnvVarImageRepoInsecure); ok {
				if result, err := strconv.ParseBool(val); err == nil {
					return result
				}
			}

			return false
		}()

		BeforeEach(func() {
			testID = generateTestID("timestamp")

			outputImage, err = name.ParseReference(fmt.Sprintf("%s/%s:%s",
				os.Getenv(EnvVarImageRepo),
				testID,
				"latest",
			))
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when using BuildKit based Dockerfile build", func() {
			var sampleBuildRun = func(outputTimestamp string) *buildv1beta1.BuildRun {
				return NewBuildRunPrototype().
					Namespace(testBuild.Namespace).
					Name(testID).
					WithBuildSpec(NewBuildPrototype().
						ClusterBuildStrategy("buildkit").
						Namespace(testBuild.Namespace).
						Name(testID).
						SourceGit("https://github.com/shipwright-io/sample-nodejs").
						SourceGitRevision("0be20591d7096bef165949c22f6059f5d8eb6a85").
						SourceContextDir("docker-build").
						Dockerfile("Dockerfile").
						OutputImage(outputImage.String()).
						OutputImageCredentials(os.Getenv(EnvVarImageRepoSecret)).
						OutputImageInsecure(insecure).
						OutputTimestamp(outputTimestamp).
						BuildSpec()).
					MustCreate()
			}

			It("should create an image with creation timestamp set to unix epoch timestamp zero", func() {
				buildRun := validateBuildRunToSucceed(testBuild, sampleBuildRun(buildv1beta1.OutputImageZeroTimestamp))
				image := testBuild.GetImage(buildRun)
				Expect(creationTimeOf(image)).To(BeTemporally("==", time.Unix(0, 0)))
			})

			It("should create an image with creation timestamp set to the source timestamp", func() {
				buildRun := validateBuildRunToSucceed(testBuild, sampleBuildRun(buildv1beta1.OutputImageSourceTimestamp))
				image := testBuild.GetImage(buildRun)
				Expect(creationTimeOf(image)).To(BeTemporally("==", time.Unix(1699261787, 0)))
			})

			It("should create an image with creation timestamp set to the build timestamp", func() {
				buildRun := validateBuildRunToSucceed(testBuild, sampleBuildRun(buildv1beta1.OutputImageBuildTimestamp))
				image := testBuild.GetImage(buildRun)
				Expect(creationTimeOf(image)).To(BeTemporally("==", buildRun.CreationTimestamp.Time))
			})

			It("should create an image with creation timestamp set to a custom timestamp", func() {
				buildRun := validateBuildRunToSucceed(testBuild, sampleBuildRun("1691691691"))
				image := testBuild.GetImage(buildRun)
				Expect(creationTimeOf(image)).To(BeTemporally("==", time.Unix(1691691691, 0)))
			})
		})

		Context("when using Buildpacks build", func() {
			var sampleBuildRun = func(outputTimestamp string) *buildv1beta1.BuildRun {
				return NewBuildRunPrototype().
					Namespace(testBuild.Namespace).
					Name(testID).
					WithBuildSpec(NewBuildPrototype().
						ClusterBuildStrategy("buildpacks-v3").
						Namespace(testBuild.Namespace).
						Name(testID).
						SourceGit("https://github.com/shipwright-io/sample-nodejs").
						SourceGitRevision("0be20591d7096bef165949c22f6059f5d8eb6a85").
						SourceContextDir("source-build").
						OutputImage(outputImage.String()).
						OutputImageCredentials(os.Getenv(EnvVarImageRepoSecret)).
						OutputImageInsecure(insecure).
						OutputTimestamp(outputTimestamp).
						Env("BP_NODE_VERSION", "~20").
						BuildSpec()).
					MustCreate()
			}

			It("should create an image with creation timestamp set to unix epoch timestamp zero", func() {
				buildRun := validateBuildRunToSucceed(testBuild, sampleBuildRun(buildv1beta1.OutputImageZeroTimestamp))
				image := testBuild.GetImage(buildRun)
				Expect(creationTimeOf(image)).To(BeTemporally("==", time.Unix(0, 0)))
			})

			It("should create an image with creation timestamp set to the source timestamp", func() {
				buildRun := validateBuildRunToSucceed(testBuild, sampleBuildRun(buildv1beta1.OutputImageSourceTimestamp))
				image := testBuild.GetImage(buildRun)
				Expect(creationTimeOf(image)).To(BeTemporally("==", time.Unix(1699261787, 0)))
			})

			It("should create an image with creation timestamp set to the build timestamp", func() {
				buildRun := validateBuildRunToSucceed(testBuild, sampleBuildRun(buildv1beta1.OutputImageBuildTimestamp))
				image := testBuild.GetImage(buildRun)
				Expect(creationTimeOf(image)).To(BeTemporally("==", buildRun.CreationTimestamp.Time))
			})

			It("should create an image with creation timestamp set to a custom timestamp", func() {
				buildRun := validateBuildRunToSucceed(testBuild, sampleBuildRun("1691691691"))
				image := testBuild.GetImage(buildRun)
				Expect(creationTimeOf(image)).To(BeTemporally("==", time.Unix(1691691691, 0)))
			})
		})

		Context("edge cases", func() {
			It("should fail run a build run when source timestamp is used with an empty source", func() {
				buildRun = NewBuildRunPrototype().
					Namespace(testBuild.Namespace).
					Name(testID).
					WithBuildSpec(NewBuildPrototype().
						ClusterBuildStrategy("buildkit").
						Namespace(testBuild.Namespace).
						Name(testID).
						OutputImage(outputImage.String()).
						OutputImageCredentials(os.Getenv(EnvVarImageRepoSecret)).
						OutputImageInsecure(insecure).
						OutputTimestamp(buildv1beta1.OutputImageSourceTimestamp).
						BuildSpec()).
					MustCreate()

				Expect(testBuild.CreateBR(buildRun)).ToNot(Succeed())

				buildRun, err = testBuild.GetBRTillCompletion(buildRun.Name)
				Expect(err).ToNot(HaveOccurred())

				condition := buildRun.Status.GetCondition(buildv1beta1.Succeeded)
				Expect(condition).ToNot(BeNil())
				Expect(condition.Reason).To(ContainSubstring("TaskRunGenerationFailed"))
				Expect(condition.Message).To(ContainSubstring("cannot use SourceTimestamp setting"))
			})

			It("should fail fail when output timestamp value is not valid", func() {
				buildRun = NewBuildRunPrototype().
					Namespace(testBuild.Namespace).
					Name(testID).
					WithBuildSpec(NewBuildPrototype().
						ClusterBuildStrategy("buildkit").
						Namespace(testBuild.Namespace).
						Name(testID).
						SourceGit("https://github.com/shipwright-io/sample-nodejs").
						SourceGitRevision("0be20591d7096bef165949c22f6059f5d8eb6a85").
						SourceContextDir("docker-build").
						Dockerfile("Dockerfile").
						OutputImage(outputImage.String()).
						OutputImageCredentials(os.Getenv(EnvVarImageRepoSecret)).
						OutputImageInsecure(insecure).
						OutputTimestamp("WrongValue").
						BuildSpec()).
					MustCreate()

				Expect(testBuild.CreateBR(buildRun)).ToNot(Succeed())

				buildRun, err = testBuild.GetBRTillCompletion(buildRun.Name)
				Expect(err).ToNot(HaveOccurred())

				condition := buildRun.Status.GetCondition(buildv1beta1.Succeeded)
				Expect(condition).ToNot(BeNil())
				Expect(condition.Reason).To(ContainSubstring("Failed"))
				Expect(condition.Message).To(ContainSubstring("cannot parse output timestamp"))
			})
		})
	})
})

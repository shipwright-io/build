// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"os"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	shpgit "github.com/shipwright-io/build/pkg/git"
)

var _ = Describe("For a Kubernetes cluster with Tekton and build installed", func() {
	var (
		testID string
		err    error

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

	Context("when a Buildah build is defined that is using shipwright-managed push", Label("FEATURE:Buildah", "CORE", "SAMPLE"), func() {

		BeforeEach(func() {
			testID = generateTestID("buildah")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1alpha1/build/build_buildah_shipwright_managed_push_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1alpha1/buildrun/buildrun_buildah_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")
			appendRegistryInsecureParamValue(build, buildRun)

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildah build is defined that is using strategy-managed push", Label("FEATURE:Buildah", "CORE", "SAMPLE"), func() {

		BeforeEach(func() {
			testID = generateTestID("buildah")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1alpha1/build/build_buildah_strategy_managed_push_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1alpha1/buildrun/buildrun_buildah_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")
			appendRegistryInsecureParamValue(build, buildRun)

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildah build with a contextDir and a custom Dockerfile name is defined", Label("FEATURE:Buildah", "CORE"), func() {

		BeforeEach(func() {
			testID = generateTestID("buildah-custom-context-dockerfile")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1alpha1/build_buildah_cr_custom_context+dockerfile.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1alpha1/buildrun_buildah_cr_custom_context+dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")
			appendRegistryInsecureParamValue(build, buildRun)

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a heroku Buildpacks build is defined using a cluster strategy", Label("FEATURE:Buildpacks", "CORE", "SAMPLE"), func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-heroku")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1alpha1/build/build_buildpacks-v3-heroku_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1alpha1/buildrun/buildrun_buildpacks-v3-heroku_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a heroku Buildpacks build is defined using a namespaced strategy", Label("FEATURE:Buildpacks", "CORE", "SAMPLE"), func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-heroku-namespaced")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1alpha1/build/build_buildpacks-v3-heroku_namespaced_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1alpha1/buildrun/buildrun_buildpacks-v3-heroku_namespaced_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})

		AfterEach(func() {
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when a Buildpacks v3 build is defined using a cluster strategy", Label("FEATURE:Buildpacks", "CORE", "SAMPLE"), func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1alpha1/build/build_buildpacks-v3_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1alpha1/buildrun/buildrun_buildpacks-v3_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildpacks v3 build is defined using a namespaced strategy", Label("FEATURE:Buildpacks", "CORE", "SAMPLE"), func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-namespaced")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1alpha1/build/build_buildpacks-v3_namespaced_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1alpha1/buildrun/buildrun_buildpacks-v3_namespaced_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})

		AfterEach(func() {
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when a Buildpacks v3 build is defined for a php runtime", Label("FEATURE:Buildpacks", "CORE"), func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-php")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1alpha1/build_buildpacks-v3_php_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1alpha1/buildrun_buildpacks-v3_php_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildpacks v3 build is defined for a ruby runtime", Label("FEATURE:Buildpacks", "CORE"), func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-ruby")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1alpha1/build_buildpacks-v3_ruby_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1alpha1/buildrun_buildpacks-v3_ruby_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildpacks v3 build is defined for a golang runtime", Label("FEATURE:Buildpacks", "CORE"), func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-golang")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1alpha1/build_buildpacks-v3_golang_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1alpha1/buildrun_buildpacks-v3_golang_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildpacks v3 build is defined for a golang runtime with `BP_GO_TARGETS` env", Label("FEATURE:Buildpacks", "CORE"), func() {
		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-golang")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1alpha1/build_buildpacks-v3_golang_cr_env.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1alpha1/buildrun_buildpacks-v3_golang_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a build uses the build-run-deletion annotation", Label("FEATURE:BuildRunDeletion", "CORE"), func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-golang")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1alpha1/build_buildpacks-v3_golang_delete_cr.yaml",
			)
		})

		It("successfully deletes the BuildRun after the Build is deleted", func() {
			By("running a build and expecting it to succeed")
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1alpha1/buildrun_buildpacks-v3_golang_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)

			By("deleting the parent Build object")
			err = testBuild.DeleteBuild(build.Name)
			Expect(err).NotTo(HaveOccurred(), "error deleting the parent Build")
			Eventually(func() bool {
				_, err = testBuild.GetBR(buildRun.Name)
				if err == nil {
					return false
				}
				if !errors.IsNotFound(err) {
					return false
				}
				return true
			}).WithTimeout(10 + time.Second).Should(BeTrue())
		})
	})

	Context("when a Buildpacks v3 build is defined for a java runtime", Label("FEATURE:Buildpacks", "CORE"), func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-java")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1alpha1/build_buildpacks-v3_java_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1alpha1/buildrun_buildpacks-v3_java_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Kaniko build is defined to use public GitHub", Label("FEATURE:Kaniko", "CORE"), func() {

		BeforeEach(func() {
			testID = generateTestID("kaniko")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1alpha1/build/build_kaniko_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1alpha1/buildrun/buildrun_kaniko_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Kaniko build with a Dockerfile that requires advanced permissions is defined", Label("FEATURE:Kaniko", "CORE"), func() {

		BeforeEach(func() {
			testID = generateTestID("kaniko-advanced-dockerfile")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1alpha1/build_kaniko_cr_advanced_dockerfile.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1alpha1/buildrun_kaniko_cr_advanced_dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Kaniko build with a contextDir and a custom Dockerfile name is defined", Label("FEATURE:Kaniko", "CORE"), func() {

		BeforeEach(func() {
			testID = generateTestID("kaniko-custom-context-dockerfile")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1alpha1/build_kaniko_cr_custom_context+dockerfile.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1alpha1/buildrun_kaniko_cr_custom_context+dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildkit build with a contextDir and a path to a Dockerfile is defined", Label("FEATURE:BuildKit", "CORE"), func() {

		BeforeEach(func() {
			testID = generateTestID("buildkit-custom-context")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1alpha1/build/build_buildkit_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1alpha1/buildrun/buildrun_buildkit_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImagePlatformsExist(buildRun, []v1.Platform{
				{
					Architecture: "amd64",
					OS:           "linux",
				},
				{
					Architecture: "arm64",
					OS:           "linux",
				},
			})
		})
	})

	Context("when a s2i build is defined", Label("FEATURE:SourceToImage", "CORE", "SAMPLE"), func() {

		BeforeEach(func() {
			testID = generateTestID("s2i")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/v1alpha1/build/build_source-to-image_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1alpha1/buildrun/buildrun_source-to-image_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a private source repository is used", Label("FEATURE:PrivateRepo"), func() {

		BeforeEach(func() {
			if os.Getenv(EnvVarEnablePrivateRepos) != "true" {
				Skip("Skipping test cases that use a private source repository")
			}
		})

		Context("when a Buildah build is defined to use a private GitHub repository", Label("FEATURE:Buildah"), func() {

			BeforeEach(func() {
				testID = generateTestID("private-github-buildah")

				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/v1alpha1/build_buildah_cr_private_github.yaml",
				)
			})

			It("successfully runs a build", func() {
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1alpha1/buildrun/buildrun_buildah_cr.yaml")
				Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

				buildRun = validateBuildRunToSucceed(testBuild, buildRun)
				testBuild.ValidateImageDigest(buildRun)
			})
		})

		Context("when a Buildah build is defined to use a private GitLab repository", Label("FEATURE:Buildah"), func() {

			BeforeEach(func() {
				testID = generateTestID("private-gitlab-buildah")

				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/v1alpha1/build_buildah_cr_private_gitlab.yaml",
				)
			})

			It("successfully runs a build", func() {
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1alpha1/buildrun/buildrun_buildah_cr.yaml")
				Expect(err).ToNot(HaveOccurred())

				buildRun = validateBuildRunToSucceed(testBuild, buildRun)
				testBuild.ValidateImageDigest(buildRun)
			})
		})

		Context("when a Kaniko build is defined to use a private GitHub repository", Label("FEATURE:Kaniko"), func() {

			BeforeEach(func() {
				testID = generateTestID("private-github-kaniko")

				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/v1alpha1/build_kaniko_cr_private_github.yaml",
				)
			})

			It("successfully runs a build", func() {
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1alpha1/buildrun/buildrun_kaniko_cr.yaml")
				Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

				buildRun = validateBuildRunToSucceed(testBuild, buildRun)
				testBuild.ValidateImageDigest(buildRun)
			})
		})

		Context("when a Kaniko build is defined to use a private GitLab repository", Label("FEATURE:Kaniko"), func() {

			BeforeEach(func() {
				testID = generateTestID("private-gitlab-kaniko")

				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/v1alpha1/build_kaniko_cr_private_gitlab.yaml",
				)
			})

			It("successfully runs a build", func() {
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1alpha1/buildrun/buildrun_kaniko_cr.yaml")
				Expect(err).ToNot(HaveOccurred())

				buildRun = validateBuildRunToSucceed(testBuild, buildRun)
				testBuild.ValidateImageDigest(buildRun)
			})
		})

		Context("when a s2i build is defined to use a private GitHub repository", Label("FEATURE:SourceToImage"), func() {

			BeforeEach(func() {
				testID = generateTestID("private-github-s2i")

				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/v1alpha1/build_source-to-image_cr_private_github.yaml",
				)
			})

			It("successfully runs a build", func() {
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/v1alpha1/buildrun/buildrun_source-to-image_cr.yaml")
				Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

				buildRun = validateBuildRunToSucceed(testBuild, buildRun)
				testBuild.ValidateImageDigest(buildRun)
			})
		})
	})

	Context("when a s2i build uses a non-existent git repository as source", Label("FEATURE:SourceToImage", "FAILING"), func() {
		It("fails because of prompted authentication which surfaces the to the BuildRun", func() {
			testID = generateTestID("s2i-failing")

			build = createBuild(
				testBuild,
				testID,
				"test/data/v1alpha1/build_non_existing_repo.yaml",
			)

			buildRun, err = buildRunTestData(build.Namespace, testID, "test/data/v1alpha1/buildrun_non_existing_repo.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToFail(testBuild, buildRun)
			buildRun, err = testBuild.LookupBuildRun(types.NamespacedName{Name: buildRun.Name, Namespace: testBuild.Namespace})

			Expect(buildRun.Status.FailureDetails.Message).To(Equal(shpgit.AuthPrompted.ToMessage()))
			Expect(buildRun.Status.FailureDetails.Reason).To(Equal(shpgit.AuthPrompted.String()))
		})
	})

})

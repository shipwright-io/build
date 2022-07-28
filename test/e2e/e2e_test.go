// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"os"

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

	Context("when a Buildah build is defined", func() {

		BeforeEach(func() {
			testID = generateTestID("buildah")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_buildah_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_buildah_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")
			appendRegistryInsecureParamValue(build, buildRun)

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildah build with a contextDir and a custom Dockerfile name is defined", func() { // build_buildah_cr_custom_context+dockerfile.yaml
		BeforeEach(func() {
			testID = generateTestID("buildah-custom-context-dockerfile")

			/*
				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/build_buildah_cr_custom_context+dockerfile.yaml",
				)*/
			build, err = NewBuildPrototype().
				ClusterBuildStrategy("buildah").
				Name(testID).
				Namespace(testBuild.Namespace).
				SourceGit("https://github.com/shipwright-io/sample-nodejs.git").
				SourceContextDir("docker-build-renamed").
				Dockerfile("RenamedDockerfile").
				OutputImage("image-registry.openshift-image-registry.svc:5000/build-examples/renamed-dockerfile").
				Create()
			Expect(err).ToNot(HaveOccurred())
		})

		It("successfully runs a build", func() {
			/*
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_buildah_cr_custom_context+dockerfile.yaml")
				Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")*/

			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				Namespace(testBuild.Namespace).
				ForBuild(build).
				Create()
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			appendRegistryInsecureParamValue(build, buildRun)

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a heroku Buildpacks build is defined using a cluster strategy", func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-heroku")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_buildpacks-v3-heroku_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_buildpacks-v3-heroku_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a heroku Buildpacks build is defined using a namespaced strategy", func() {
		var buildStrategy *buildv1alpha1.BuildStrategy

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-heroku-namespaced")

			buildStrategy, err = buildStrategyTestData(testBuild.Namespace, "samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_namespaced_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			err = testBuild.CreateBuildStrategy(buildStrategy)
			Expect(err).ToNot(HaveOccurred())

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_buildpacks-v3-heroku_namespaced_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_buildpacks-v3-heroku_namespaced_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})

		AfterEach(func() {
			err = testBuild.DeleteBuildStrategy(buildStrategy.Name)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when a Buildpacks v3 build is defined using a cluster strategy", func() {

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_buildpacks-v3_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_buildpacks-v3_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildpacks v3 build is defined using a namespaced strategy", func() {
		var buildStrategy *buildv1alpha1.BuildStrategy

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-namespaced")

			buildStrategy, err = buildStrategyTestData(testBuild.Namespace, "samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_namespaced_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			err = testBuild.CreateBuildStrategy(buildStrategy)
			Expect(err).ToNot(HaveOccurred())

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_buildpacks-v3_namespaced_cr.yaml",
			)
		})

		It("successfully runs a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_buildpacks-v3_namespaced_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})

		AfterEach(func() {
			err = testBuild.DeleteBuildStrategy(buildStrategy.Name)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when a Buildpacks v3 build is defined for a php runtime", func() { // build_buildpacks-v3_php_cr.yaml

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-php")
			/*
				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/build_buildpacks-v3_php_cr.yaml",
				)*/

			build, err = NewBuildPrototype().
				ClusterBuildStrategy("buildpacks-v3").
				Name(testID).
				Namespace(testBuild.Namespace).
				SourceGit("https://github.com/shipwright-io/sample-php.git").
				SourceContextDir("source-build").
				OutputImage("image-registry.openshift-image-registry.svc:5000/build-examples/taxi-app").
				Create()
			Expect(err).ToNot(HaveOccurred())
		})

		It("successfully runs a build", func() {
			/*buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_buildpacks-v3_php_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")*/

			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				Namespace(testBuild.Namespace).
				ForBuild(build).
				GenerateServiceAccount().
				Create()
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildpacks v3 build is defined for a ruby runtime", func() { // build_buildpacks-v3_ruby_cr.yaml

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-ruby")
			/*
				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/build_buildpacks-v3_ruby_cr.yaml",
				)*/

			build, err = NewBuildPrototype().
				ClusterBuildStrategy("buildpacks-v3").
				Name(testID).
				Namespace(testBuild.Namespace).
				SourceGit("https://github.com/shipwright-io/sample-ruby.git").
				SourceContextDir("source-build").
				OutputImage("image-registry.openshift-image-registry.svc:5000/build-examples/taxi-app").
				Create()
			Expect(err).ToNot(HaveOccurred())
		})

		It("successfully runs a build", func() {
			/*buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_buildpacks-v3_ruby_cr.yaml")
			Expect(err).ToNot(HaveOccurred())*/
			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				Namespace(testBuild.Namespace).
				ForBuild(build).
				GenerateServiceAccount().
				Create()
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildpacks v3 build is defined for a golang runtime", func() { // build_buildpacks-v3_golang_cr.yaml

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-golang")
			/*
				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/build_buildpacks-v3_golang_cr.yaml",
				)*/

			build, err = NewBuildPrototype().
				ClusterBuildStrategy("buildpacks-v3").
				Name(testID).
				Namespace(testBuild.Namespace).
				SourceGit("https://github.com/shipwright-io/sample-go.git").
				SourceContextDir("source-build").
				OutputImage("image-registry.openshift-image-registry.svc:5000/build-examples/taxi-app").
				Create()
			Expect(err).ToNot(HaveOccurred())
		})

		It("successfully runs a build", func() {
			/*buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_buildpacks-v3_golang_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")*/

			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				Namespace(testBuild.Namespace).
				ForBuild(build).
				GenerateServiceAccount().
				Create()
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Buildpacks v3 build is defined for a golang runtime with `BP_GO_TARGETS` env", func() { // build_buildpacks-v3_golang_cr_env.yaml
		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-golang")

			/*
				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/build_buildpacks-v3_golang_cr_env.yaml",
				)*/

			build, err = NewBuildPrototype().
				ClusterBuildStrategy("buildpacks-v3").
				Name(testID).
				Namespace(testBuild.Namespace).
				SourceGit("https://github.com/shipwright-io/sample-go.git").
				SourceContextDir("source-build-with-package").
				ArrayParamValue("BP_GO_TARGETS", "\"main-package\""). // name is BP_GO_TARGETS", value is "main-package"
				OutputImage("image-registry.openshift-image-registry.svc:5000/build-examples/taxi-app").
				Create()
			Expect(err).ToNot(HaveOccurred())
		})

		It("successfully runs a build", func() {
			/*buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_buildpacks-v3_golang_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")*/
			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				Namespace(testBuild.Namespace).
				ForBuild(build).
				GenerateServiceAccount().
				Create()
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a build uses the build-run-deletion annotation", func() { // build_buildpacks-v3_golang_delete_cr.yaml

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-golang")
			/*
				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/build_buildpacks-v3_golang_delete_cr.yaml",
				)*/
			build, err = NewBuildPrototype().
				ClusterBuildStrategy("buildpacks-v3").
				Name(testID).
				Namespace(testBuild.Namespace).
				Annotations(map[string]string{"build.shipwright.io/build-run-deletion": "\"true\""}).
				SourceGit("https://github.com/shipwright-io/sample-go.git").
				SourceContextDir("source-build").
				OutputImage("image-registry.openshift-image-registry.svc:5000/build-examples/taxi-app").
				Create()
			Expect(err).ToNot(HaveOccurred())
		})

		It("successfully deletes the BuildRun after the Build is deleted", func() {
			By("running a build and expecting it to succeed")
			/*buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_buildpacks-v3_golang_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")*/
			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				Namespace(testBuild.Namespace).
				ForBuild(build).
				GenerateServiceAccount().
				Create()
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
			}).Should(BeTrue())
		})
	})

	Context("when a Buildpacks v3 build is defined for a java runtime", func() { // build_buildpacks-v3_java_cr.yaml

		BeforeEach(func() {
			testID = generateTestID("buildpacks-v3-java")
			/*
				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/build_buildpacks-v3_java_cr.yaml",
				)*/

			build, err = NewBuildPrototype().
				ClusterBuildStrategy("buildpacks-v3").
				Name(testID).
				Namespace(testBuild.Namespace).
				SourceGit("https://github.com/shipwright-io/sample-java.git").
				SourceContextDir("source-build").
				OutputImage("image-registry.openshift-image-registry.svc:5000/build-examples/taxi-app").
				Create()
			Expect(err).ToNot(HaveOccurred())
		})

		It("successfully runs a build", func() {
			/*buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_buildpacks-v3_java_cr.yaml")
			Expect(err).ToNot(HaveOccurred())*/

			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				Namespace(testBuild.Namespace).
				ForBuild(build).
				GenerateServiceAccount().
				Create()
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Kaniko build is defined to use public GitHub", func() {

		BeforeEach(func() {
			testID = generateTestID("kaniko")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_kaniko_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_kaniko_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Kaniko build with a Dockerfile that requires advanced permissions is defined", func() { // build_kaniko_cr_advanced_dockerfile.yaml

		BeforeEach(func() {
			testID = generateTestID("kaniko-advanced-dockerfile")
			/*
				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/build_kaniko_cr_advanced_dockerfile.yaml",
				)*/

			build, err = NewBuildPrototype().
				ClusterBuildStrategy("kaniko").
				Name(testID).
				Namespace(testBuild.Namespace).
				Annotations(map[string]string{"build.shipwright.io/build-run-deletion": "\"false\""}).
				SourceGit("https://github.com/shipwright-io/sample-java.git").
				SourceContextDir("docker-build").
				Dockerfile("Dockerfile").
				OutputImage("image-registry.openshift-image-registry.svc:5000/build-examples/advanced-dockerfile").
				Create()
			Expect(err).ToNot(HaveOccurred())
		})

		It("successfully runs a build", func() {
			/*buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_kaniko_cr_advanced_dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")*/

			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				Namespace(testBuild.Namespace).
				ForBuild(build).
				Create()
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Kaniko build with a contextDir and a custom Dockerfile name is defined", func() { // build_kaniko_cr_custom_context+dockerfile.yaml

		BeforeEach(func() {
			testID = generateTestID("kaniko-custom-context-dockerfile")
			/*
				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/build_kaniko_cr_custom_context+dockerfile.yaml",
				)*/
			build, err = NewBuildPrototype().
				ClusterBuildStrategy("kaniko").
				Name(testID).
				Namespace(testBuild.Namespace).
				SourceGit("https://github.com/shipwright-io/sample-nodejs.git").
				SourceContextDir("docker-build").
				Dockerfile("Dockerfile").
				OutputImage("image-registry.openshift-image-registry.svc:5000/build-examples/renamed-dockerfile").
				Create()
			Expect(err).ToNot(HaveOccurred())
		})

		It("successfully runs a build", func() {
			/*buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/buildrun_kaniko_cr_custom_context+dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred())*/

			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				Namespace(testBuild.Namespace).
				ForBuild(build).
				Create()
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Kaniko+Trivy build is defined to use an image with no critical CVEs", func() {

		BeforeEach(func() {
			testID = generateTestID("kaniko-trivy-good")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_kaniko-trivy-good_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_kaniko-trivy-good_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a Kaniko+Trivy build is defined to use an image with a critical CVE", func() {

		BeforeEach(func() {
			testID = generateTestID("kaniko-trivy-bad")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_kaniko-trivy-bad_cr.yaml",
			)
		})

		It("fails to run a build", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_kaniko-trivy-bad_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToFail(testBuild, buildRun)
		})
	})

	Context("when a Buildkit build with a contextDir and a path to a Dockerfile is defined", func() { // build_buildkit_cr_insecure_registry.yaml

		BeforeEach(func() {
			testID = generateTestID("buildkit-custom-context")
			/*
				// create the build definition
				build = createBuild(
					testBuild,
					testID,
					"test/data/build_buildkit_cr_insecure_registry.yaml",
				)*/
			build, err = NewBuildPrototype().
				ClusterBuildStrategy("buildkit").
				Name(testID).
				Namespace(testBuild.Namespace).
				Annotations(map[string]string{"build.build.dev/build-run-deletion": "\"true'"}).
				SourceGit("https://github.com/shipwright-io/sample-go.git").
				SourceContextDir("docker-build").
				ArrayParamValue("insecure-registry", "\"true\"").
				ArrayParamValue("platforms", "linux/amd64").
				ArrayParamValue("platforms", "linux/arm64").
				OutputImage("image-registry.openshift-image-registry.svc:5000/build-examples/taxi-app").
				Create()
			Expect(err).ToNot(HaveOccurred())

		})

		It("successfully runs a build and surface results to BuildRun", func() {
			/*
				buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_buildkit_cr.yaml")
				Expect(err).ToNot(HaveOccurred())*/

			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				Namespace(testBuild.Namespace).
				ForBuild(build).
				Create()
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

	Context("when a s2i build is defined", func() {

		BeforeEach(func() {
			testID = generateTestID("s2i")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"samples/build/build_source-to-image_cr.yaml",
			)
		})

		It("successfully runs a build and surface results to BuildRun", func() {
			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_source-to-image_cr.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})
	})

	Context("when a private source repository is used", func() {

		BeforeEach(func() {
			if os.Getenv(EnvVarEnablePrivateRepos) != "true" {
				Skip("Skipping test cases that use a private source repository")
			}
		})

		Context("when a Buildah build is defined to use a private GitHub repository", func() { // build_buildah_cr_private_github.yaml

			BeforeEach(func() {
				testID = generateTestID("private-github-buildah")

				/*
					// create the build definition
					build = createBuild(
						testBuild,
						testID,
						"test/data/build_buildah_cr_private_github.yaml",
					)*/

				build, err = NewBuildPrototype().
					ClusterBuildStrategy("buildah").
					Name(testID).
					Namespace(testBuild.Namespace).
					SourceGit("git@github.com:qu1queee/newtaxi.git").
					SourceCredentials("github-ssh-all").
					Dockerfile("Dockerfile").
					OutputImage("image-registry.openshift-image-registry.svc:5000/build-examples/taxi-app").
					Create()
				Expect(err).ToNot(HaveOccurred())

			})

			It("successfully runs a build", func() {
				/*
					buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_buildah_cr.yaml")
					Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")*/

				buildRun, err = NewBuildRunPrototype().
					Name(testID).
					Namespace(testBuild.Namespace).
					ForBuild(build).
					Create()
				Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

				buildRun = validateBuildRunToSucceed(testBuild, buildRun)
				testBuild.ValidateImageDigest(buildRun)
			})
		})

		Context("when a Buildah build is defined to use a private GitLab repository", func() { // build_buildah_cr_private_gitlab.yaml

			BeforeEach(func() {
				testID = generateTestID("private-gitlab-buildah")
				/*
					// create the build definition
					build = createBuild(
						testBuild,
						testID,
						"test/data/build_buildah_cr_private_gitlab.yaml",
					)*/
				build, err = NewBuildPrototype().
					ClusterBuildStrategy("buildah").
					Name(testID).
					Namespace(testBuild.Namespace).
					SourceGit("git@gitlab.com:eduardooli/newtaxi.git").
					Dockerfile("Dockerfile").
					OutputImage("image-registry.openshift-image-registry.svc:5000/build-examples/taxi-app").
					Create()
				Expect(err).ToNot(HaveOccurred())
			})

			It("successfully runs a build", func() {
				/*
					buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_buildah_cr.yaml")
					Expect(err).ToNot(HaveOccurred())*/
				buildRun, err = NewBuildRunPrototype().
					Name(testID).
					Namespace(testBuild.Namespace).
					ForBuild(build).
					Create()
				Expect(err).ToNot(HaveOccurred())

				buildRun = validateBuildRunToSucceed(testBuild, buildRun)
				testBuild.ValidateImageDigest(buildRun)
			})
		})

		Context("when a Kaniko build is defined to use a private GitHub repository", func() { // build_kaniko_cr_private_github.yaml

			BeforeEach(func() {
				testID = generateTestID("private-github-kaniko")
				/*
					// create the build definition
					build = createBuild(
						testBuild,
						testID,
						"test/data/build_kaniko_cr_private_github.yaml",
					)*/

				build, err = NewBuildPrototype().
					ClusterBuildStrategy("kaniko").
					Name(testID).
					Namespace(testBuild.Namespace).
					SourceGit("git@github.com:qu1queee/newtaxi.git").
					Dockerfile("Dockerfile").
					OutputImage("image-registry.openshift-image-registry.svc:5000/build-examples/taxi-app").
					Create()
				Expect(err).ToNot(HaveOccurred())
			})

			It("successfully runs a build", func() {
				/*buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_kaniko_cr.yaml")
				Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")*/

				buildRun, err = NewBuildRunPrototype().
					Name(testID).
					Namespace(testBuild.Namespace).
					ForBuild(build).
					Create()
				Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

				buildRun = validateBuildRunToSucceed(testBuild, buildRun)
				testBuild.ValidateImageDigest(buildRun)
			})
		})

		Context("when a Kaniko build is defined to use a private GitLab repository", func() { // build_kaniko_cr_private_gitlab.yaml

			BeforeEach(func() {
				testID = generateTestID("private-gitlab-kaniko")
				/*
					// create the build definition
					build = createBuild(
						testBuild,
						testID,
						"test/data/build_kaniko_cr_private_gitlab.yaml",
					)*/

				build, err = NewBuildPrototype().
					ClusterBuildStrategy("kaniko").
					Name(testID).
					Namespace(testBuild.Namespace).
					SourceGit("git@gitlab.com:eduardooli/newtaxi.git").
					Dockerfile("Dockerfile").
					OutputImage("image-registry.openshift-image-registry.svc:5000/build-examples/taxi-app").
					Create()
				Expect(err).ToNot(HaveOccurred())
			})

			It("successfully runs a build", func() {
				/*buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_kaniko_cr.yaml")
				Expect(err).ToNot(HaveOccurred())*/

				buildRun, err = NewBuildRunPrototype().
					Name(testID).
					Namespace(testBuild.Namespace).
					ForBuild(build).
					Create()
				Expect(err).ToNot(HaveOccurred())

				buildRun = validateBuildRunToSucceed(testBuild, buildRun)
				testBuild.ValidateImageDigest(buildRun)
			})
		})

		Context("when a s2i build is defined to use a private GitHub repository", func() { // build_source-to-image_cr_private_github.yaml

			BeforeEach(func() {
				testID = generateTestID("private-github-s2i")
				/*
					// create the build definition
					build = createBuild(
						testBuild,
						testID,
						"test/data/build_source-to-image_cr_private_github.yaml",
					)*/

				build, err = NewBuildPrototype().
					ClusterBuildStrategy("source-to-image").
					Name(testID).
					Namespace(testBuild.Namespace).
					SourceGit("git@github.com:shipwright-io/sample-nodejs-private.git").
					BuilderImage("docker.io/centos/nodejs-10-centos7").
					OutputImage("image-registry.openshift-image-registry.svc:5000/build-examples/nodejs-ex").
					Create()
				Expect(err).ToNot(HaveOccurred())
			})

			It("successfully runs a build", func() {
				/*buildRun, err = buildRunTestData(testBuild.Namespace, testID, "samples/buildrun/buildrun_source-to-image_cr.yaml")
				Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")*/

				buildRun, err = NewBuildRunPrototype().
					Name(testID).
					Namespace(testBuild.Namespace).
					ForBuild(build).
					Create()
				Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

				buildRun = validateBuildRunToSucceed(testBuild, buildRun)
				testBuild.ValidateImageDigest(buildRun)
			})
		})
	})

	Context("when a s2i build uses a non-existent git repository as source", func() { // build_non_existing_repo.yaml
		It("fails because of prompted authentication which surfaces the to the BuildRun", func() {
			testID = generateTestID("s2i-failing")
			/*
				build = createBuild(
					testBuild,
					testID,
					"test/data/build_non_existing_repo.yaml",
				)*/

			build, err = NewBuildPrototype().
				ClusterBuildStrategy("kaniko").
				Name(testID).
				Namespace(testBuild.Namespace).
				SourceGit("https://github.com/shipwright-io/sample-nodejs-no-exists.git").
				OutputImage("image-registry.openshift-image-registry.svc:5000/build-examples/no-exists").
				Create()
			Expect(err).ToNot(HaveOccurred())

			/*buildRun, err = buildRunTestData(build.Namespace, testID, "test/data/buildrun_non_existing_repo.yaml")
			Expect(err).ToNot(HaveOccurred())*/

			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				Namespace(testBuild.Namespace).
				ForBuild(build).
				Create()
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToFail(testBuild, buildRun)
			buildRun, err = testBuild.LookupBuildRun(types.NamespacedName{Name: buildRun.Name, Namespace: testBuild.Namespace})

			Expect(buildRun.Status.FailureDetails.Message).To(Equal(shpgit.AuthPrompted.ToMessage()))
			Expect(buildRun.Status.FailureDetails.Reason).To(Equal(shpgit.AuthPrompted.String()))
		})
	})

})

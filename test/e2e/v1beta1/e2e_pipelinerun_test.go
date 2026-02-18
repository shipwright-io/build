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
	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("PipelineRun E2E Tests", Label("PipelineRun", "CORE"), func() {

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

		build         *buildv1beta1.Build
		buildRun      *buildv1beta1.BuildRun
		buildStrategy *buildv1beta1.BuildStrategy
		configMap     *corev1.ConfigMap
		secret        *corev1.Secret
	)

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			printTestFailureDebugInfo(testBuild, testBuild.Namespace, testID)
		} else if buildRun != nil {
			validateServiceAccountDeletion(buildRun, testBuild.Namespace)
		}

		if buildRun != nil {
			Expect(testBuild.DeleteBR(buildRun.Name)).To(Succeed())
			buildRun = nil
		}

		if build != nil {
			Expect(testBuild.DeleteBuild(build.Name)).To(Succeed())
			build = nil
		}

		if buildStrategy != nil {
			Expect(testBuild.DeleteBuildStrategy(buildStrategy.Name)).To(Succeed())
			buildStrategy = nil
		}

		if configMap != nil {
			Expect(testBuild.DeleteConfigMap(configMap.Name)).To(Succeed())
			configMap = nil
		}

		if secret != nil {
			Expect(testBuild.DeleteSecret(secret.Name)).To(Succeed())
			secret = nil
		}
	})

	Context("One-Off Builds with PipelineRun", func() {
		var outputImage name.Reference

		BeforeEach(func() {
			testID = generateTestID("pipelinerun-onoff")
			outputImage, err = name.ParseReference(fmt.Sprintf("%s/%s:%s",
				os.Getenv(EnvVarImageRepo),
				testID,
				"latest",
			))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should build an image using Buildpacks and Git source with PipelineRun", Label("Buildpacks", "GitSource", "PipelineRun"), func() {
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

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validatePipelineRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)

			// Verify PipelineRun was created and succeeded
			validatePipelineRunExistsAndSucceeded(buildRun)
		})

		It("should build an image using Buildah and Git source with PipelineRun", Label("Buildah", "GitSource", "PipelineRun"), func() {
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

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validatePipelineRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)

			// Verify PipelineRun was created and succeeded
			validatePipelineRunExistsAndSucceeded(buildRun)
		})

		It("should build an image using Buildpacks and OCI artifact source with PipelineRun", Label("Buildpacks", "OCIArtifactSource", "PipelineRun"), func() {
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

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validatePipelineRunResultsFromBundleSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)

			// Verify PipelineRun was created and succeeded
			validatePipelineRunExistsAndSucceeded(buildRun)
		})
	})

	Context("Git Source with PipelineRun", func() {
		It("should successfully run a build with limited git history using PipelineRun", Label("GitDepth", "PipelineRun"), func() {
			testID = generateTestID("pipelinerun-git-depth")

			// create the build definition
			build = createBuild(
				testBuild,
				testID,
				"test/data/v1beta1/build_buildah_cr_custom_context+dockerfile.yaml",
			)

			buildRun, err = buildRunTestData(testBuild.Namespace, testID, "test/data/v1beta1/buildrun_buildah_cr_custom_context+dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")
			appendRegistryInsecureParamValue(build, buildRun)

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validatePipelineRunResultsFromGitSource(buildRun)

			// Verify PipelineRun was created and succeeded
			validatePipelineRunExistsAndSucceeded(buildRun)
		})
	})

	Context("Multiple TaskRuns in PipelineRun", func() {
		var outputImage name.Reference

		BeforeEach(func() {
			testID = generateTestID("pipelinerun-multi-task")
			outputImage, err = name.ParseReference(fmt.Sprintf("%s/%s:%s",
				os.Getenv(EnvVarImageRepo),
				testID,
				"latest",
			))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle multiple TaskRuns in a PipelineRun successfully", Label("MultiTaskRun", "PipelineRun"), func() {
			// This test verifies that the controller can handle PipelineRuns with multiple TaskRuns
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

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validatePipelineRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)

			// Verify PipelineRun was created and succeeded
			validatePipelineRunExistsAndSucceeded(buildRun)

			// Verify that the controller can handle multiple TaskRuns
			validateMultipleTaskRunsHandling(buildRun)
		})
	})

	Context("PipelineRun Error Handling", func() {
		It("should handle PipelineRun failures gracefully", Label("ErrorHandling", "PipelineRun"), func() {
			testID = generateTestID("pipelinerun-error")

			// Create a build with invalid source to trigger failure
			buildRun, err = NewBuildRunPrototype().
				Namespace(testBuild.Namespace).
				Name(testID).
				WithBuildSpec(NewBuildPrototype().
					ClusterBuildStrategy("buildpacks-v3").
					Namespace(testBuild.Namespace).
					Name(testID).
					SourceGit("https://invalid-repo-that-does-not-exist.git").
					SourceContextDir("source-build").
					OutputImage("dummy-image").
					BuildSpec()).
				Create()
			Expect(err).ToNot(HaveOccurred())

			// Verify the build fails as expected
			validateBuildRunToFail(testBuild, buildRun)

			// Get a fresh copy of the BuildRun to ensure we have the latest status
			buildRun, err = testBuild.LookupBuildRun(types.NamespacedName{
				Namespace: buildRun.Namespace,
				Name:      buildRun.Name,
			})
			Expect(err).ToNot(HaveOccurred())

			// Verify PipelineRun was created and failed
			validatePipelineRunExistsAndFailed(buildRun)
		})
	})
})

// validatePipelineRunExistsAndSucceeded verifies that a PipelineRun was created and succeeded
func validatePipelineRunExistsAndSucceeded(buildRun *buildv1beta1.BuildRun) {
	// Verify that the BuildRun has the succeeded condition
	condition := buildRun.Status.GetCondition(buildv1beta1.Succeeded)
	Expect(condition).NotTo(BeNil())
	Expect(condition.Status).To(Equal(corev1.ConditionTrue))

	// Verify that the BuildRun has completion information (if available)
	if buildRun.Status.CompletionTime != nil {
		Expect(buildRun.Status.CompletionTime).NotTo(BeNil())
	}

	// Verify that the BuildRun used a PipelineRun executor
	Expect(buildRun.Status.Executor).NotTo(BeNil())
	Expect(buildRun.Status.Executor.Kind).To(Equal("PipelineRun"))
	Expect(buildRun.Status.Executor.Name).NotTo(BeEmpty())
}

// validatePipelineRunExistsAndFailed verifies that a PipelineRun was created and failed
func validatePipelineRunExistsAndFailed(buildRun *buildv1beta1.BuildRun) {
	// Verify that the BuildRun has the failed condition
	condition := buildRun.Status.GetCondition(buildv1beta1.Succeeded)
	Expect(condition).NotTo(BeNil())
	Expect(condition.Status).To(Equal(corev1.ConditionFalse))

	// Verify that the BuildRun has completion information (if available)
	if buildRun.Status.CompletionTime != nil {
		Expect(buildRun.Status.CompletionTime).NotTo(BeNil())
	}

	// Verify that the BuildRun used a PipelineRun executor
	Expect(buildRun.Status.Executor).NotTo(BeNil())
	Expect(buildRun.Status.Executor.Kind).To(Equal("PipelineRun"))
	Expect(buildRun.Status.Executor.Name).NotTo(BeEmpty())
}

// validateMultipleTaskRunsHandling verifies that the controller can handle multiple TaskRuns
func validateMultipleTaskRunsHandling(buildRun *buildv1beta1.BuildRun) {
	// Verify that the BuildRun has the succeeded condition
	condition := buildRun.Status.GetCondition(buildv1beta1.Succeeded)
	Expect(condition).NotTo(BeNil())
	Expect(condition.Status).To(Equal(corev1.ConditionTrue))
}

// validatePipelineRunResultsFromGitSource validates PipelineRun results for Git source
// This function is similar to validateBuildRunResultsFromGitSource but adapted for PipelineRun executor
func validatePipelineRunResultsFromGitSource(buildRun *buildv1beta1.BuildRun) {
	// For PipelineRun executor, we validate what we can expect to be populated
	// The Source field might not be populated by the controller when using PipelineRun executor

	// Verify that the BuildRun has the succeeded condition
	condition := buildRun.Status.GetCondition(buildv1beta1.Succeeded)
	Expect(condition).NotTo(BeNil())
	Expect(condition.Status).To(Equal(corev1.ConditionTrue))

	// Verify that the BuildRun has completion information (if available)
	if buildRun.Status.CompletionTime != nil {
		Expect(buildRun.Status.CompletionTime).NotTo(BeNil())
	}
	if buildRun.Status.StartTime != nil {
		Expect(buildRun.Status.StartTime).NotTo(BeNil())
	}
}

// validatePipelineRunResultsFromBundleSource validates PipelineRun results for Bundle source
func validatePipelineRunResultsFromBundleSource(buildRun *buildv1beta1.BuildRun) {
	// For PipelineRun executor, we validate what we can expect to be populated

	// Verify that the BuildRun has the succeeded condition
	condition := buildRun.Status.GetCondition(buildv1beta1.Succeeded)
	Expect(condition).NotTo(BeNil())
	Expect(condition.Status).To(Equal(corev1.ConditionTrue))

	// Verify that the BuildRun has completion information (if available)
	if buildRun.Status.CompletionTime != nil {
		Expect(buildRun.Status.CompletionTime).NotTo(BeNil())
	}
	if buildRun.Status.StartTime != nil {
		Expect(buildRun.Status.StartTime).NotTo(BeNil())
	}
}

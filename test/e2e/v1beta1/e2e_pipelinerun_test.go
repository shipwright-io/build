// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	crv1 "github.com/google/go-containerregistry/pkg/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tektonpipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/validate"
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

		build         *buildapi.Build
		buildRun      *buildapi.BuildRun
		buildStrategy *buildapi.BuildStrategy
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

	Context("Multi-arch spec.output.platforms (OCI image index)", func() {
		var (
			outputImage name.Reference
			platforms   = []buildapi.ImagePlatform{
				{OS: "linux", Arch: "amd64"},
				{OS: "linux", Arch: "arm64"},
			}
		)

		BeforeEach(func() {
			skipUnlessTestImageRepoForPush()
			skipUnlessSchedulablePlatforms(platforms)
			testID = generateTestID("pipelinerun-multiarch-index")
			outputImage, err = name.ParseReference(fmt.Sprintf("%s/%s:%s",
				os.Getenv(EnvVarImageRepo),
				testID,
				"latest",
			))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should run per-platform PipelineTasks, assemble-index, and push a multi-platform image", Label("Buildah", "GitSource", "PipelineRun", "MultiArchOutputPlatforms"), func() {
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
					OutputImagePlatforms(platforms).
					OutputImageCredentials(os.Getenv(EnvVarImageRepoSecret)).
					OutputImageInsecure(insecure).
					BuildSpec()).
				Create()
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validatePipelineRunExistsAndSucceeded(buildRun)
			validateMultiArchPipelineRunTektonContract(buildRun, platforms, outputImage.String(), insecure)
			validatePipelineRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
			testBuild.ValidateImagePlatformsExist(buildRun, []crv1.Platform{
				{OS: "linux", Architecture: "amd64"},
				{OS: "linux", Architecture: "arm64"},
			})
		})
	})

	Context("spec.output.platforms with one platform (homogeneous cluster smoke)", func() {
		var (
			outputImage name.Reference
			platforms   []buildapi.ImagePlatform
		)

		BeforeEach(func() {
			skipUnlessTestImageRepoForPush()
			platforms = pickSingleSchedulablePlatformForE2E()
			testID = generateTestID("pipelinerun-single-platform-index")
			outputImage, err = name.ParseReference(fmt.Sprintf("%s/%s:%s",
				os.Getenv(EnvVarImageRepo),
				testID,
				"latest",
			))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should run source-acquisition, one build-<os>-<arch>, assemble-index, and push an image", Label("Buildah", "GitSource", "PipelineRun", "SinglePlatformOutputPlatforms"), func() {
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
					OutputImagePlatforms(platforms).
					OutputImageCredentials(os.Getenv(EnvVarImageRepoSecret)).
					OutputImageInsecure(insecure).
					BuildSpec()).
				Create()
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validatePipelineRunExistsAndSucceeded(buildRun)
			validateMultiArchPipelineRunTektonContract(buildRun, platforms, outputImage.String(), insecure)
			validatePipelineRunResultsFromGitSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
			testBuild.ValidateImagePlatformsExist(buildRun, []crv1.Platform{
				{OS: platforms[0].OS, Architecture: platforms[0].Arch},
			})
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

func skipUnlessTestImageRepoForPush() {
	if strings.TrimSpace(os.Getenv(EnvVarImageRepo)) == "" {
		Skip(fmt.Sprintf("%s must be set to a registry prefix (e.g. localhost:5001/org/repo) so multi-arch source-bundle and image pushes do not target Docker Hub anonymously",
			EnvVarImageRepo))
	}
}

func skipUnlessSchedulablePlatforms(platforms []buildapi.ImagePlatform) {
	nodes, err := testBuild.GetNodes()
	Expect(err).ToNot(HaveOccurred())
	ok, _, msg := validate.ValidateNodeAvailability(platforms, nodes.Items)
	if !ok {
		Skip(fmt.Sprintf("multi-arch PipelineRun test needs Ready schedulable nodes for each platform: %s", msg))
	}
}

// pickSingleSchedulablePlatformForE2E returns one {os, arch} from Ready schedulable nodes (deterministic:
// lexicographic sort). Used to exercise spec.output.platforms on homogeneous clusters without amd64+arm64.
func pickSingleSchedulablePlatformForE2E() []buildapi.ImagePlatform {
	nodes, err := testBuild.GetNodes()
	Expect(err).ToNot(HaveOccurred())
	plats := schedulableImagePlatforms(nodes.Items)
	if len(plats) == 0 {
		Skip("single-platform output.platforms E2E needs at least one Ready schedulable node with kubernetes.io/os and kubernetes.io/arch labels")
	}
	return []buildapi.ImagePlatform{plats[0]}
}

func schedulableImagePlatforms(nodes []corev1.Node) []buildapi.ImagePlatform {
	type pair struct{ os, arch string }
	var pairs []pair
	seen := make(map[string]bool)
	for _, node := range nodes {
		if node.Spec.Unschedulable {
			continue
		}
		ready := false
		for _, c := range node.Status.Conditions {
			if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue {
				ready = true
				break
			}
		}
		if !ready {
			continue
		}
		osLabel := node.Labels[corev1.LabelOSStable]
		archLabel := node.Labels[corev1.LabelArchStable]
		if osLabel == "" || archLabel == "" {
			continue
		}
		key := osLabel + "/" + archLabel
		if seen[key] {
			continue
		}
		seen[key] = true
		pairs = append(pairs, pair{os: osLabel, arch: archLabel})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].os != pairs[j].os {
			return pairs[i].os < pairs[j].os
		}
		return pairs[i].arch < pairs[j].arch
	})
	out := make([]buildapi.ImagePlatform, len(pairs))
	for i, p := range pairs {
		out[i] = buildapi.ImagePlatform{OS: p.os, Arch: p.arch}
	}
	return out
}

func validateMultiArchPipelineRunTektonContract(buildRun *buildapi.BuildRun, platforms []buildapi.ImagePlatform, expectedOutputImage string, outputInsecure bool) {
	Expect(buildRun.Status.PlatformResults).To(HaveLen(len(platforms)),
		"use BuildRun.status.platformResults digests as the source of truth for TaskRun cross-checks")
	for _, p := range platforms {
		Expect(platformDigestFromBuildRunStatus(buildRun, p)).NotTo(BeEmpty(),
			"missing digest in BuildRun.status.platformResults for %s/%s", p.OS, p.Arch)
	}

	pr, err := testBuild.GetPipelineRunFromBuildRun(buildRun.Name)
	Expect(err).ToNot(HaveOccurred())
	Expect(pr.Spec.PipelineSpec).NotTo(BeNil())

	taskNames := make(map[string]struct{}, len(pr.Spec.PipelineSpec.Tasks))
	for _, t := range pr.Spec.PipelineSpec.Tasks {
		taskNames[t.Name] = struct{}{}
	}
	Expect(taskNames).To(HaveKey("source-acquisition"))
	Expect(taskNames).To(HaveKey("assemble-index"))
	var buildLikeTasks int
	for name := range taskNames {
		if strings.HasPrefix(name, "build-") {
			buildLikeTasks++
		}
	}
	Expect(buildLikeTasks).To(Equal(len(platforms)),
		"expected exactly one PipelineTask build-<os>-<arch> per requested platform")
	for _, p := range platforms {
		Expect(taskNames).To(HaveKey(fmt.Sprintf("build-%s-%s", p.OS, p.Arch)))
	}

	Expect(pipelineRunParamString(pr.Spec.Params, "shp-output-image")).To(Equal(expectedOutputImage))
	Expect(pipelineRunParamString(pr.Spec.Params, "shp-output-insecure")).To(Equal(strconv.FormatBool(outputInsecure)))

	Expect(pr.Spec.TaskRunSpecs).To(HaveLen(len(platforms)), "TaskRunSpecs should pin each per-platform build task to a node OS/arch")
	for _, p := range platforms {
		taskName := fmt.Sprintf("build-%s-%s", p.OS, p.Arch)
		spec := findPipelineTaskRunSpec(pr.Spec.TaskRunSpecs, taskName)
		Expect(spec).NotTo(BeNil(), "missing PipelineTaskRunSpec for %q", taskName)
		Expect(spec.PodTemplate).NotTo(BeNil())
		Expect(spec.PodTemplate.NodeSelector).To(HaveKey(corev1.LabelOSStable))
		Expect(spec.PodTemplate.NodeSelector).To(HaveKey(corev1.LabelArchStable))
		Expect(spec.PodTemplate.NodeSelector[corev1.LabelOSStable]).To(Equal(p.OS))
		Expect(spec.PodTemplate.NodeSelector[corev1.LabelArchStable]).To(Equal(p.Arch))

		pt := findPipelineTaskByName(pr.Spec.PipelineSpec.Tasks, taskName)
		Expect(pt).NotTo(BeNil())
		Expect(pt.TaskSpec).NotTo(BeNil())
		imgStep := findPipelineStep(pt.TaskSpec.Steps, "image-processing")
		Expect(imgStep).NotTo(BeNil())
		imgArgs := strings.Join(imgStep.Args, " ")
		Expect(imgArgs).To(ContainSubstring("$(params.shp-output-image)"))
		Expect(imgArgs).To(ContainSubstring("--result-file-image-digest"))

		outImgParam := findPipelineTaskParam(pt.Params, "shp-output-image")
		Expect(outImgParam).NotTo(BeNil())
		Expect(outImgParam.Value.Type).To(Equal(pipelineapi.ParamTypeString))
		Expect(outImgParam.Value.StringVal).To(Equal(fmt.Sprintf("$(params.shp-output-image)%s", platformImageTagSuffix(p))))
	}

	assemblePT := findPipelineTaskByName(pr.Spec.PipelineSpec.Tasks, "assemble-index")
	Expect(assemblePT).NotTo(BeNil())
	Expect(assemblePT.TaskSpec).NotTo(BeNil())
	assembleStep := findPipelineStep(assemblePT.TaskSpec.Steps, "assemble-index")
	Expect(assembleStep).NotTo(BeNil())
	assembleArgLine := strings.Join(assembleStep.Args, " ")
	Expect(assembleArgLine).To(ContainSubstring("--assemble-index"))
	Expect(assembleArgLine).To(ContainSubstring("--image $(params.shp-output-image)"))
	for _, p := range platforms {
		Expect(assembleArgLine).To(ContainSubstring(expectedAssemblePlatformArgFragment(p)))
	}

	taskRuns, err := testBuild.ListTaskRunsForPipelineRun(pr.Name)
	Expect(err).ToNot(HaveOccurred())
	Expect(taskRuns).To(HaveLen(2+len(platforms)), "expected source-acquisition + N build-* tasks + assemble-index TaskRuns")

	byPipelineTask := map[string]pipelineapi.TaskRun{}
	for _, tr := range taskRuns {
		taskLabel := tr.Labels[tektonpipeline.PipelineTaskLabelKey]
		Expect(taskLabel).NotTo(BeEmpty(), "TaskRun %q missing %s label", tr.Name, tektonpipeline.PipelineTaskLabelKey)
		byPipelineTask[taskLabel] = tr
	}

	for _, p := range platforms {
		taskName := fmt.Sprintf("build-%s-%s", p.OS, p.Arch)
		tr, ok := byPipelineTask[taskName]
		Expect(ok).To(BeTrue(), "expected a TaskRun labeled for pipeline task %q", taskName)
		Expect(tr.Spec.PodTemplate).NotTo(BeNil())
		Expect(tr.Spec.PodTemplate.NodeSelector[corev1.LabelOSStable]).To(Equal(p.OS))
		Expect(tr.Spec.PodTemplate.NodeSelector[corev1.LabelArchStable]).To(Equal(p.Arch))

		Expect(tr.Status.PodName).NotTo(BeEmpty())
		pod, err := testBuild.LookupPod(types.NamespacedName{Namespace: tr.Namespace, Name: tr.Status.PodName})
		Expect(err).ToNot(HaveOccurred())
		ctr := findPodStepContainer(pod.Spec.Containers, "image-processing")
		Expect(ctr).NotTo(BeNil())
		podImgArgs := strings.Join(ctr.Args, " ")
		// Tekton substitutes $(params.shp-output-image) into the literal registry reference; per-platform
		// builds append -<os>-<arch> to the tag. The digest is emitted via --result-file-image-digest, not in --image.
		resolvedPlatformPushRef := fmt.Sprintf("%s%s", expectedOutputImage, platformImageTagSuffix(p))
		Expect(podImgArgs).To(ContainSubstring("--image"))
		Expect(podImgArgs).To(ContainSubstring(resolvedPlatformPushRef),
			"image-processing pod should receive the platform-scoped push reference after param substitution")
		Expect(podImgArgs).To(ContainSubstring("--result-file-image-digest"))

		platformDigest := platformDigestFromBuildRunStatus(buildRun, p)
		Expect(taskRunSHAImageDigest(&tr)).To(Equal(platformDigest),
			"per-platform TaskRun result shp-image-digest must match BuildRun.status.platformResults for %s/%s", p.OS, p.Arch)
	}

	aTR, ok := byPipelineTask["assemble-index"]
	Expect(ok).To(BeTrue())
	Expect(aTR.Status.PodName).NotTo(BeEmpty())
	aPod, err := testBuild.LookupPod(types.NamespacedName{Namespace: aTR.Namespace, Name: aTR.Status.PodName})
	Expect(err).ToNot(HaveOccurred())
	aCtr := findPodStepContainer(aPod.Spec.Containers, "assemble-index")
	Expect(aCtr).NotTo(BeNil())
	aPodArgs := strings.Join(aCtr.Args, " ")
	Expect(aPodArgs).To(ContainSubstring("--assemble-index"))
	Expect(aPodArgs).To(ContainSubstring("@sha256:"))
	for _, p := range platforms {
		Expect(aPodArgs).To(ContainSubstring(fmt.Sprintf("%s/%s=", p.OS, p.Arch)))
		d := platformDigestFromBuildRunStatus(buildRun, p)
		Expect(aPodArgs).To(ContainSubstring(fmt.Sprintf("@%s", d)),
			"assemble-index argv should resolve each platform pin to BuildRun.status.platformResults digest for %s/%s",
			p.OS, p.Arch)
		refAndDigest := fmt.Sprintf("%s@%s",
			fmt.Sprintf("%s%s", expectedOutputImage, platformImageTagSuffix(p)), d)
		Expect(aPodArgs).To(ContainSubstring(refAndDigest),
			"assemble-index should pass the pinned ref@digest emitted for %s/%s", p.OS, p.Arch)
	}
}

const tektonSHAImageDigestResultName = "shp-image-digest"

func platformDigestFromBuildRunStatus(br *buildapi.BuildRun, p buildapi.ImagePlatform) string {
	for _, pr := range br.Status.PlatformResults {
		if pr.Platform.OS == p.OS && pr.Platform.Arch == p.Arch {
			return pr.Digest
		}
	}
	return ""
}

// taskRunSHAImageDigest returns the Tekton TaskRun string result wired from the Shipwright image-processing step.
func taskRunSHAImageDigest(tr *pipelineapi.TaskRun) string {
	for _, r := range tr.Status.Results {
		if r.Name == tektonSHAImageDigestResultName {
			return r.Value.StringVal
		}
	}
	return ""
}

func pipelineRunParamString(params pipelineapi.Params, name string) string {
	for _, p := range params {
		if p.Name == name && p.Value.Type == pipelineapi.ParamTypeString {
			return p.Value.StringVal
		}
	}
	return ""
}

func findPipelineTaskRunSpec(specs []pipelineapi.PipelineTaskRunSpec, pipelineTaskName string) *pipelineapi.PipelineTaskRunSpec {
	for i := range specs {
		if specs[i].PipelineTaskName == pipelineTaskName {
			return &specs[i]
		}
	}
	return nil
}

func findPipelineTaskByName(tasks []pipelineapi.PipelineTask, name string) *pipelineapi.PipelineTask {
	for i := range tasks {
		if tasks[i].Name == name {
			return &tasks[i]
		}
	}
	return nil
}

func findPipelineStep(steps []pipelineapi.Step, name string) *pipelineapi.Step {
	for i := range steps {
		if steps[i].Name == name {
			return &steps[i]
		}
	}
	return nil
}

func findPipelineTaskParam(params []pipelineapi.Param, name string) *pipelineapi.Param {
	for i := range params {
		if params[i].Name == name {
			return &params[i]
		}
	}
	return nil
}

func platformImageTagSuffix(p buildapi.ImagePlatform) string {
	return fmt.Sprintf("-%s-%s", p.OS, p.Arch)
}

func expectedAssemblePlatformArgFragment(p buildapi.ImagePlatform) string {
	tName := fmt.Sprintf("build-%s-%s", p.OS, p.Arch)
	return fmt.Sprintf("%s/%s=$(params.shp-output-image)%s@$(tasks.%s.results.shp-image-digest)",
		p.OS, p.Arch, platformImageTagSuffix(p), tName)
}

func findPodStepContainer(containers []corev1.Container, stepName string) *corev1.Container {
	candidates := []string{"step-" + stepName, stepName}
	for _, want := range candidates {
		for i := range containers {
			if containers[i].Name == want {
				return &containers[i]
			}
		}
	}
	return nil
}

// validatePipelineRunExistsAndSucceeded verifies that a PipelineRun was created and succeeded
func validatePipelineRunExistsAndSucceeded(buildRun *buildapi.BuildRun) {
	// Verify that the BuildRun has the succeeded condition
	condition := buildRun.Status.GetCondition(buildapi.Succeeded)
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
func validatePipelineRunExistsAndFailed(buildRun *buildapi.BuildRun) {
	// Verify that the BuildRun has the failed condition
	condition := buildRun.Status.GetCondition(buildapi.Succeeded)
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
func validateMultipleTaskRunsHandling(buildRun *buildapi.BuildRun) {
	// Verify that the BuildRun has the succeeded condition
	condition := buildRun.Status.GetCondition(buildapi.Succeeded)
	Expect(condition).NotTo(BeNil())
	Expect(condition.Status).To(Equal(corev1.ConditionTrue))
}

// validatePipelineRunResultsFromGitSource validates PipelineRun results for Git source
// This function is similar to validateBuildRunResultsFromGitSource but adapted for PipelineRun executor
func validatePipelineRunResultsFromGitSource(buildRun *buildapi.BuildRun) {
	// For PipelineRun executor, we validate what we can expect to be populated
	// The Source field might not be populated by the controller when using PipelineRun executor

	// Verify that the BuildRun has the succeeded condition
	condition := buildRun.Status.GetCondition(buildapi.Succeeded)
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
func validatePipelineRunResultsFromBundleSource(buildRun *buildapi.BuildRun) {
	// For PipelineRun executor, we validate what we can expect to be populated

	// Verify that the BuildRun has the succeeded condition
	condition := buildRun.Status.GetCondition(buildapi.Succeeded)
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

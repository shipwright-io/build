// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"
	"maps"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/sources"
)

const (
	sourceBundleTagSuffix  = "-src"
	paramSourceTimestamp   = "source-timestamp"
	paramSourceBundleImage = "source-bundle-image"
)

// platformTaskName returns the PipelineTask name for a given platform build.
func platformTaskName(p buildapi.ImagePlatform) string {
	return fmt.Sprintf("build-%s-%s", p.OS, p.Arch)
}

// platformImageTag returns the tag suffix for a per-platform image.
func platformImageTag(p buildapi.ImagePlatform) string {
	return fmt.Sprintf("-%s-%s", p.OS, p.Arch)
}

// sourceBundleImageParam returns the Tekton parameter expression for the
// source bundle OCI artifact reference (base output image + "-src" suffix).
func sourceBundleImageParam() string {
	return fmt.Sprintf("$(params.%s-%s)%s", prefixParamsResultsVolumes, paramOutputImage, sourceBundleTagSuffix)
}

// effectivePushSecret returns the push secret to use for output image operations.
// BuildRun.spec.output.pushSecret takes precedence over Build.spec.output.pushSecret
// when a BuildRun-level override is provided.
func effectivePushSecret(build *buildapi.Build, buildRun *buildapi.BuildRun) *string {
	if buildRun.Spec.Output != nil && buildRun.Spec.Output.PushSecret != nil {
		return buildRun.Spec.Output.PushSecret
	}
	return build.Spec.Output.PushSecret
}

// generateSourceBundlePushStep creates a step that packages the workspace source
// directory as an OCI artifact and pushes it to the registry. If a pushSecret is
// provided, the secret volume is added to the taskSpec and mounted into the step
// so the image-processing binary can authenticate with the registry.
func generateSourceBundlePushStep(cfg *config.Config, taskSpec *pipelineapi.TaskSpec, pushSecret *string) pipelineapi.Step {
	args := []string{
		"--push-source-bundle", fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramSourceRoot),
		"--source-bundle-image", sourceBundleImageParam(),
		fmt.Sprintf("--insecure=$(params.%s-%s)", prefixParamsResultsVolumes, paramOutputInsecure),
	}

	var volumeMounts []corev1.VolumeMount
	if pushSecret != nil {
		sources.AppendSecretVolume(taskSpec, *pushSecret)
		secretMountPath := fmt.Sprintf("/workspace/%s-push-secret", prefixParamsResultsVolumes)
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      sources.SanitizeVolumeNameForSecretName(*pushSecret),
			MountPath: secretMountPath,
			ReadOnly:  true,
		})
		args = append(args, "--secret-path", secretMountPath)
	}

	step := pipelineapi.Step{
		Name:             "push-source-bundle",
		Image:            cfg.ImageProcessingContainerTemplate.Image,
		ImagePullPolicy:  cfg.ImageProcessingContainerTemplate.ImagePullPolicy,
		Command:          cfg.ImageProcessingContainerTemplate.Command,
		Args:             args,
		Env:              cfg.ImageProcessingContainerTemplate.Env,
		ComputeResources: cfg.ImageProcessingContainerTemplate.Resources,
		SecurityContext:  cfg.ImageProcessingContainerTemplate.SecurityContext,
		WorkingDir:       cfg.ImageProcessingContainerTemplate.WorkingDir,
		VolumeMounts:     volumeMounts,
	}
	sources.SetupHomeAndTmpVolumes(taskSpec, &step)
	return step
}

// createPerPlatformBuildTask generates a PipelineTask that
// pulls source from the OCI artifact,
// runs the build strategy steps, and
// pushes the result with a platform-specific tag suffix.
func createPerPlatformBuildTask(
	cfg *config.Config,
	build *buildapi.Build,
	buildRun *buildapi.BuildRun,
	strategy buildapi.BuilderStrategy,
	platform buildapi.ImagePlatform,
	execCtx *executionContext,
) (pipelineapi.PipelineTask, bool, error) {
	taskName := platformTaskName(platform)
	taskSpec := createBaseTaskSpec()

	// Add source-bundle-image param for the bundle pull step
	prefixedSourceBundleImage := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramSourceBundleImage)
	taskSpec.Params = append(taskSpec.Params, pipelineapi.ParamSpec{
		Name: prefixedSourceBundleImage,
		Type: pipelineapi.ParamTypeString,
	})

	// Accept source timestamp from source-acquisition when SourceTimestamp is requested
	buildRunOutput := buildRun.Spec.Output
	if buildRunOutput == nil {
		buildRunOutput = &buildapi.Image{}
	}
	needsSourceTimestamp := false
	if ts := getImageTimestamp(build.Spec.Output, *buildRunOutput); ts != nil && *ts == buildapi.OutputImageSourceTimestamp {
		needsSourceTimestamp = true
		taskSpec.Params = append(taskSpec.Params, pipelineapi.ParamSpec{
			Name: paramSourceTimestamp,
			Type: pipelineapi.ParamTypeString,
		})
	}

	// Step 1: Pull source from OCI artifact using the bundle binary.
	// When source is an OCI artifact, the pull secret comes from the source
	// spec; otherwise it's the output push secret (source bundle is pushed
	// to the same registry as the output image).
	bundlePullArgs := []string{
		"--image", fmt.Sprintf("$(params.%s)", prefixedSourceBundleImage),
		"--target", fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramSourceRoot),
	}
	var bundlePullVolumeMounts []corev1.VolumeMount
	pullSecret := effectivePushSecret(build, buildRun)
	if build.Spec.Source != nil && build.Spec.Source.Type == buildapi.OCIArtifactType && build.Spec.Source.OCIArtifact != nil && build.Spec.Source.OCIArtifact.PullSecret != nil {
		pullSecret = build.Spec.Source.OCIArtifact.PullSecret
	}
	if pullSecret != nil {
		sources.AppendSecretVolume(taskSpec, *pullSecret)
		secretMountPath := fmt.Sprintf("/workspace/%s-pull-secret", prefixParamsResultsVolumes)
		bundlePullVolumeMounts = append(bundlePullVolumeMounts, corev1.VolumeMount{
			Name:      sources.SanitizeVolumeNameForSecretName(*pullSecret),
			MountPath: secretMountPath,
			ReadOnly:  true,
		})
		bundlePullArgs = append(bundlePullArgs, "--secret-path", secretMountPath)
	}
	bundlePullStep := pipelineapi.Step{
		Name:             "pull-source-bundle",
		Image:            cfg.BundleContainerTemplate.Image,
		ImagePullPolicy:  cfg.BundleContainerTemplate.ImagePullPolicy,
		Command:          cfg.BundleContainerTemplate.Command,
		Args:             bundlePullArgs,
		Env:              cfg.BundleContainerTemplate.Env,
		ComputeResources: cfg.BundleContainerTemplate.Resources,
		SecurityContext:  cfg.BundleContainerTemplate.SecurityContext,
		WorkingDir:       cfg.BundleContainerTemplate.WorkingDir,
		VolumeMounts:     bundlePullVolumeMounts,
	}
	sources.SetupHomeAndTmpVolumes(taskSpec, &bundlePullStep)
	taskSpec.Steps = append(taskSpec.Steps, bundlePullStep)

	// Step 2: Build strategy steps
	hasOutputDir, err := applyBuildStrategyToTaskSpec(taskSpec, build, buildRun, strategy, execCtx)
	if err != nil {
		return pipelineapi.PipelineTask{}, false, fmt.Errorf("applying build strategy for %s: %w", taskName, err)
	}

	// Step 3: Image processing — push (if output-directory), mutate, and record digest/size
	imgProcArgs, err := buildPerPlatformImageProcessingArgs(
		cfg,
		build,
		buildRun,
		hasOutputDir,
	)
	if err != nil {
		return pipelineapi.PipelineTask{}, false, fmt.Errorf("building image processing args for %s: %w", taskName, err)
	}

	if err := CreateImageProcessingStep(
		cfg,
		taskSpec,
		imgProcArgs,
		false,
		effectivePushSecret(build, buildRun),
	); err != nil {
		return pipelineapi.PipelineTask{}, false, fmt.Errorf("creating image processing step for %s: %w", taskName, err)
	}

	// Build the task params: base params + strategy params + source bundle image
	params := generateBaseTaskParamReferences()

	// Step 3: Override shp-output-image to the platform-specific tag
	for i, p := range params {
		if p.Name == fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputImage) {
			params[i].Value = pipelineapi.ParamValue{
				Type:      pipelineapi.ParamTypeString,
				StringVal: fmt.Sprintf("$(params.%s-%s)%s", prefixParamsResultsVolumes, paramOutputImage, platformImageTag(platform)),
			}
			break
		}
	}

	sourceBundleImage := sourceBundleImageParam()
	if build.Spec.Source != nil && build.Spec.Source.Type == buildapi.OCIArtifactType && build.Spec.Source.OCIArtifact != nil {
		sourceBundleImage = build.Spec.Source.OCIArtifact.Image
	}
	params = append(params, pipelineapi.Param{
		Name: prefixedSourceBundleImage,
		Value: pipelineapi.ParamValue{
			Type:      pipelineapi.ParamTypeString,
			StringVal: sourceBundleImage,
		},
	})

	params = append(params, strategyParamReferences(strategy.GetParameters())...)

	if hasOutputDir {
		prefixedOutputDirectory := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputDirectory)
		params = append(params, pipelineapi.Param{
			Name: prefixedOutputDirectory,
			Value: pipelineapi.ParamValue{
				Type:      pipelineapi.ParamTypeString,
				StringVal: fmt.Sprintf("$(params.%s)", prefixedOutputDirectory),
			},
		})
	}

	if needsSourceTimestamp {
		params = append(params, pipelineapi.Param{
			Name: paramSourceTimestamp,
			Value: pipelineapi.ParamValue{
				Type:      pipelineapi.ParamTypeString,
				StringVal: fmt.Sprintf("$(tasks.source-acquisition.results.%s)", sources.TaskResultName(defaultSourceName, sourceTimestampName)),
			},
		})
	}

	// Step 4: Create the pipeline task
	pipelineTask := pipelineapi.PipelineTask{
		Name: taskName,
		TaskSpec: &pipelineapi.EmbeddedTask{
			TaskSpec: *taskSpec,
		},
		Params: params,
		Workspaces: []pipelineapi.WorkspacePipelineTaskBinding{
			{Name: workspaceSource, Workspace: workspaceSource},
			{Name: workspaceCache, Workspace: workspaceCache},
		},
		RunAfter: []string{"source-acquisition"},
	}

	return pipelineTask, hasOutputDir, nil
}

// createIndexAssemblyTask generates a PipelineTask that assembles an OCI image
// index from the per-platform build results.
func createIndexAssemblyTask(
	cfg *config.Config,
	platforms []buildapi.ImagePlatform,
	build *buildapi.Build,
	buildRun *buildapi.BuildRun,
) pipelineapi.PipelineTask {
	taskSpec := createBaseTaskSpec()

	// The assemble-index task only needs output-image and output-insecure params;
	// it doesn't touch source files, so strip source-root, source-context,
	// the source workspace, and the size/vulnerabilities results.
	outputImageParam := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputImage)
	outputInsecureParam := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputInsecure)
	trimmedParams := taskSpec.Params[:0]
	for _, p := range taskSpec.Params {
		if p.Name == outputImageParam || p.Name == outputInsecureParam {
			trimmedParams = append(trimmedParams, p)
		}
	}
	taskSpec.Params = trimmedParams

	taskSpec.Workspaces = nil

	digestResult := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, imageDigestResult)
	trimmedResults := taskSpec.Results[:0]
	for _, r := range taskSpec.Results {
		if r.Name == digestResult {
			trimmedResults = append(trimmedResults, r)
		}
	}
	taskSpec.Results = trimmedResults

	var platformImageArgs []string
	var runAfter []string

	for _, p := range platforms {
		tName := platformTaskName(p)
		runAfter = append(runAfter, tName)

		// Reference the per-platform image by its tag + digest from the build task result
		platformImageArgs = append(platformImageArgs,
			"--platform-image",
			fmt.Sprintf("%s/%s=$(params.%s-%s)%s@$(tasks.%s.results.%s-%s)",
				p.OS, p.Arch,
				prefixParamsResultsVolumes, paramOutputImage, platformImageTag(p),
				tName,
				prefixParamsResultsVolumes, imageDigestResult,
			),
		)
	}

	args := []string{"--assemble-index"}
	args = append(args, platformImageArgs...)
	args = append(args,
		"--image", fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramOutputImage),
		fmt.Sprintf("--insecure=$(params.%s-%s)", prefixParamsResultsVolumes, paramOutputInsecure),
		"--result-file-image-digest", fmt.Sprintf("$(results.%s-%s.path)", prefixParamsResultsVolumes, imageDigestResult),
	)

	var secretVolumeMounts []corev1.VolumeMount
	if pushSecret := effectivePushSecret(build, buildRun); pushSecret != nil {
		sources.AppendSecretVolume(taskSpec, *pushSecret)
		secretMountPath := fmt.Sprintf("/workspace/%s-push-secret", prefixParamsResultsVolumes)
		secretVolumeMounts = append(secretVolumeMounts, corev1.VolumeMount{
			Name:      sources.SanitizeVolumeNameForSecretName(*pushSecret),
			MountPath: secretMountPath,
			ReadOnly:  true,
		})
		args = append(args, "--secret-path", secretMountPath)
	}

	assemblyStep := pipelineapi.Step{
		Name:             "assemble-index",
		Image:            cfg.ImageProcessingContainerTemplate.Image,
		ImagePullPolicy:  cfg.ImageProcessingContainerTemplate.ImagePullPolicy,
		Command:          cfg.ImageProcessingContainerTemplate.Command,
		Args:             args,
		Env:              cfg.ImageProcessingContainerTemplate.Env,
		ComputeResources: cfg.ImageProcessingContainerTemplate.Resources,
		SecurityContext:  cfg.ImageProcessingContainerTemplate.SecurityContext,
		WorkingDir:       cfg.ImageProcessingContainerTemplate.WorkingDir,
		VolumeMounts:     secretVolumeMounts,
	}

	sources.SetupHomeAndTmpVolumes(taskSpec, &assemblyStep)
	taskSpec.Steps = append(taskSpec.Steps, assemblyStep)

	assemblyParams := []pipelineapi.Param{
		{Name: outputImageParam, Value: pipelineapi.ParamValue{Type: pipelineapi.ParamTypeString, StringVal: fmt.Sprintf("$(params.%s)", outputImageParam)}},
		{Name: outputInsecureParam, Value: pipelineapi.ParamValue{Type: pipelineapi.ParamTypeString, StringVal: fmt.Sprintf("$(params.%s)", outputInsecureParam)}},
	}

	return pipelineapi.PipelineTask{
		Name: "assemble-index",
		TaskSpec: &pipelineapi.EmbeddedTask{
			TaskSpec: *taskSpec,
		},
		Params:   assemblyParams,
		RunAfter: runAfter,
	}
}

// buildPerPlatformImageProcessingArgs constructs image-processing arguments
// for a per-platform build task. Unlike the single-arch path, per-platform
// tasks always need the image-processing step to record the digest and size
// as task results — the assemble-index task depends on these.
func buildPerPlatformImageProcessingArgs(
	cfg *config.Config,
	build *buildapi.Build,
	buildRun *buildapi.BuildRun,
	hasOutputDirectory bool,
) ([]string, error) {
	buildRunOutput := buildRun.Spec.Output
	if buildRunOutput == nil {
		buildRunOutput = &buildapi.Image{}
	}

	// SourceTimestamp is handled via a pipeline parameter from source-acquisition,
	// not a same-task result file, so strip it before generating args.
	// Deep-copy the Image structs so MergeMaps inside BuildImageProcessingArgs
	// does not mutate the informer-cached Build object's Annotations/Labels maps.
	buildOutput := build.Spec.Output
	buildOutput.Annotations = cloneStringMap(buildOutput.Annotations)
	buildOutput.Labels = cloneStringMap(buildOutput.Labels)
	effectiveBROutput := *buildRunOutput
	var wantsSourceTimestamp bool
	if ts := getImageTimestamp(buildOutput, effectiveBROutput); ts != nil && *ts == buildapi.OutputImageSourceTimestamp {
		wantsSourceTimestamp = true
		buildOutput.Timestamp = nil
		effectiveBROutput.Timestamp = nil
	}

	stepArgs, err := BuildImageProcessingArgs(
		cfg,
		buildRun.CreationTimestamp.Time,
		buildOutput,
		effectiveBROutput,
		hasOutputDirectory,
		false,
	)
	if err != nil {
		return nil, err
	}

	if len(stepArgs) == 0 {
		stepArgs = buildMinimalImageProcessingArgs(hasOutputDirectory)
	}

	if wantsSourceTimestamp {
		stepArgs = append(stepArgs, "--image-timestamp", fmt.Sprintf("$(params.%s)", paramSourceTimestamp))
	}

	return stepArgs, nil
}

// buildMinimalImageProcessingArgs returns the minimum set of args needed for
// the image-processing binary to load (or push from output-dir), record the
// digest and size, and exit. Used when no mutations are configured.
func buildMinimalImageProcessingArgs(hasOutputDirectory bool) []string {
	var args []string
	if hasOutputDirectory {
		args = append(args, "--push", fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramOutputDirectory))
	}
	args = append(args,
		"--image", fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramOutputImage),
		fmt.Sprintf("--insecure=$(params.%s-%s)", prefixParamsResultsVolumes, paramOutputInsecure),
		"--result-file-image-digest", fmt.Sprintf("$(results.%s-%s.path)", prefixParamsResultsVolumes, imageDigestResult),
		"--result-file-image-size", fmt.Sprintf("$(results.%s-%s.path)", prefixParamsResultsVolumes, imageSizeResult),
	)
	return args
}

// EffectiveOutputPlatforms returns the platform list to use for a BuildRun.
// BuildRun.Spec.Output.Platforms overrides the build-level platforms when non-empty.
// buildOutputPlatforms should come from the Build (at generation time) or from
// BuildRun.Status.BuildSpec (at reconciliation time).
func EffectiveOutputPlatforms(buildRun *buildapi.BuildRun, buildOutputPlatforms []buildapi.ImagePlatform) []buildapi.ImagePlatform {
	if buildRun.Spec.Output != nil && len(buildRun.Spec.Output.Platforms) > 0 {
		return buildRun.Spec.Output.Platforms
	}
	return buildOutputPlatforms
}

// validateOutputImageForMultiArch rejects digest-pinned output image references
// for multi-arch builds.
func validateOutputImageForMultiArch(build *buildapi.Build, buildRun *buildapi.BuildRun) error {
	image := build.Spec.Output.Image
	if buildRun.Spec.Output != nil && buildRun.Spec.Output.Image != "" {
		image = buildRun.Spec.Output.Image
	}
	lastSlash := strings.LastIndex(image, "/")
	nameAndTag := image
	if lastSlash >= 0 {
		nameAndTag = image[lastSlash+1:]
	}
	if strings.Contains(nameAndTag, "@") {
		return fmt.Errorf("multi-arch builds do not support digest-pinned output images (%s); use a tag reference instead", image)
	}
	return nil
}

func cloneStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	return maps.Clone(m)
}

// generateMultiArchTaskRunSpecs creates per-task PipelineTaskRunSpec entries
// with nodeSelector for each platform, merged with user-provided scheduling.
func generateMultiArchTaskRunSpecs(
	platforms []buildapi.ImagePlatform,
	baseNodeSelector map[string]string,
	baseTolerations []corev1.Toleration,
) []pipelineapi.PipelineTaskRunSpec {
	var specs []pipelineapi.PipelineTaskRunSpec
	for _, p := range platforms {
		// Clone the full base selector and overwrite os/arch keys so any
		// user-provided os/arch values are replaced by the per-platform values.
		ns := maps.Clone(baseNodeSelector)
		if ns == nil {
			ns = make(map[string]string, 2)
		}
		ns[corev1.LabelOSStable] = p.OS
		ns[corev1.LabelArchStable] = p.Arch

		podTemplate := &pod.PodTemplate{
			NodeSelector: ns,
		}
		if len(baseTolerations) > 0 {
			podTemplate.Tolerations = baseTolerations
		}

		specs = append(specs, pipelineapi.PipelineTaskRunSpec{
			PipelineTaskName: platformTaskName(p),
			PodTemplate:      podTemplate,
		})
	}

	return specs
}

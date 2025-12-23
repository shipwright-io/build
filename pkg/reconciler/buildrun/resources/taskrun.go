// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/env"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/steps"
	"github.com/shipwright-io/build/pkg/volumes"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const (
	prefixParamsResultsVolumes = "shp"

	paramOutputImage    = "output-image"
	paramOutputInsecure = "output-insecure"
	paramSourceRoot     = "source-root"
	paramSourceContext  = "source-context"

	workspaceSource = "source"
)

// GenerateTaskRun creates a Tekton TaskRun to be used for a build run
func GenerateTaskRun(
	cfg *config.Config,
	build *buildv1beta1.Build,
	buildRun *buildv1beta1.BuildRun,
	serviceAccountName string,
	strategy buildv1beta1.BuilderStrategy,
) (*pipelineapi.TaskRun, error) {
	// Generate base TaskRun with TaskSpec
	taskRun, err := initializeTaskRun(cfg, build, buildRun, serviceAccountName, strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize TaskRun for BuildRun %q using strategy %q: %w",
			buildRun.Name, strategy.GetName(), err)
	}

	if err := applyNodeSelectors(taskRun, build, buildRun); err != nil {
		return nil, fmt.Errorf("failed to apply node selectors to TaskRun for BuildRun %q: %w", buildRun.Name, err)
	}

	if err := applyTolerations(taskRun, build, buildRun); err != nil {
		return nil, fmt.Errorf("failed to apply tolerations to TaskRun for BuildRun %q: %w", buildRun.Name, err)
	}

	if err := applyScheduler(taskRun, build, buildRun); err != nil {
		return nil, fmt.Errorf("failed to apply scheduler configuration to TaskRun for BuildRun %q: %w", buildRun.Name, err)
	}

	if err := applyRuntimeClassName(taskRun, build, buildRun); err != nil {
		return nil, fmt.Errorf("failed to apply runtimeClassName to TaskRun for BuildRun %q: %w", buildRun.Name, err)
	}

	if err := applyAnnotationsAndLabels(taskRun, strategy); err != nil {
		return nil, fmt.Errorf("failed to apply annotations and labels to TaskRun for BuildRun %q using strategy %q: %w", buildRun.Name, strategy.GetName(), err)
	}

	if err := applyTimeout(taskRun, build, buildRun); err != nil {
		return nil, fmt.Errorf("failed to apply timeout configuration to TaskRun for BuildRun %q: %w", buildRun.Name, err)
	}

	if err := applyParameters(taskRun, build, buildRun, strategy); err != nil {
		return nil, fmt.Errorf("failed to apply parameters to TaskRun for BuildRun %q using strategy %q: %w", buildRun.Name, strategy.GetName(), err)
	}

	if err := applyOutputImageSteps(cfg, taskRun, build, buildRun); err != nil {
		return nil, fmt.Errorf("failed to apply output image processing steps to TaskRun for BuildRun %q: %w", buildRun.Name, err)
	}

	return taskRun, nil
}

// initializeTaskRun creates the base TaskRun with TaskSpec and basic metadata
func initializeTaskRun(cfg *config.Config, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun, serviceAccountName string, strategy buildv1beta1.BuilderStrategy) (*pipelineapi.TaskRun, error) {
	taskSpec, err := buildTaskSpec(cfg, build, buildRun, strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to build TaskSpec for BuildRun %q using strategy %q: %w",
			buildRun.Name, strategy.GetName(), err)
	}

	taskRun := &pipelineapi.TaskRun{
		ObjectMeta: generateTaskRunMetadata(build, buildRun),
		Spec: pipelineapi.TaskRunSpec{
			ServiceAccountName: serviceAccountName,
			TaskSpec:           taskSpec,
			Workspaces:         generateWorkspaceBindings(),
		},
	}

	return taskRun, nil
}

// buildTaskSpec creates a complete TaskSpec by orchestrating all TaskSpec building functions
func buildTaskSpec(cfg *config.Config, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun, strategy buildv1beta1.BuilderStrategy) (*pipelineapi.TaskSpec, error) {
	taskSpec := createBaseTaskSpec()

	applySourcesToTaskSpec(cfg, taskSpec, build, buildRun)
	addStrategyParametersToTaskSpec(taskSpec, strategy.GetParameters())

	combinedEnvs, err := mergeEnvironmentVariables(build, buildRun)
	if err != nil {
		return nil, fmt.Errorf("failed to merge environment variables for BuildRun %q: %w",
			buildRun.Name, err)
	}

	volumeMounts, err := applyBuildStrategySteps(taskSpec, build, strategy.GetBuildSteps(), strategy.GetVolumes(), combinedEnvs)
	if err != nil {
		return nil, fmt.Errorf("failed to apply build strategy steps for strategy %q: %w", strategy.GetName(), err)
	}

	if err := generateTaskSpecVolumes(taskSpec, volumeMounts, strategy.GetVolumes(), build.Spec.Volumes, buildRun.Spec.Volumes); err != nil {
		return nil, fmt.Errorf("failed to generate TaskSpec volumes for strategy %q: %w", strategy.GetName(), err)
	}

	return taskSpec, nil
}

// createBaseTaskSpec creates the initial TaskSpec with base components
func createBaseTaskSpec() *pipelineapi.TaskSpec {
	return &pipelineapi.TaskSpec{
		Params:     generateBaseTaskSpecParams(),
		Workspaces: generateBaseTaskSpecWorkspaces(),
		Results:    generateBaseTaskSpecResults(),
	}
}

// generateTaskRunMetadata creates the ObjectMeta for the TaskRun
func generateTaskRunMetadata(build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		GenerateName: buildRun.Name + "-",
		Namespace:    buildRun.Namespace,
		Labels:       generateTaskRunLabels(build, buildRun),
	}
}

// generateWorkspaceBindings creates the workspace bindings for the TaskRun
func generateWorkspaceBindings() []pipelineapi.WorkspaceBinding {
	return []pipelineapi.WorkspaceBinding{
		{
			Name:     workspaceSource,
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

// applyBuildStrategySteps processes build strategy steps and adds them to TaskSpec with environment variable handling
func applyBuildStrategySteps(
	taskSpec *pipelineapi.TaskSpec,
	build *buildv1beta1.Build,
	buildSteps []buildv1beta1.Step,
	buildStrategyVolumes []buildv1beta1.BuildStrategyVolume,
	combinedEnvs []corev1.EnvVar,
) (map[string]bool, error) {
	volumeMounts := make(map[string]bool)
	buildStrategyVolumesMap := toVolumeMap(buildStrategyVolumes)

	for _, containerValue := range buildSteps {
		// Merge environment variables for this step
		stepEnv, err := env.MergeEnvVars(combinedEnvs, containerValue.Env, false)
		if err != nil {
			return nil, fmt.Errorf("error(s) occurred merging environment variables into BuildStrategy %q steps: %s", build.Spec.StrategyName(), err.Error())
		}

		step := pipelineapi.Step{
			Image:            containerValue.Image,
			ImagePullPolicy:  containerValue.ImagePullPolicy,
			Name:             containerValue.Name,
			VolumeMounts:     containerValue.VolumeMounts,
			Command:          containerValue.Command,
			Args:             containerValue.Args,
			SecurityContext:  containerValue.SecurityContext,
			WorkingDir:       containerValue.WorkingDir,
			ComputeResources: containerValue.Resources,
			Env:              stepEnv,
		}

		taskSpec.Steps = append(taskSpec.Steps, step)

		// Validate and collect volume mounts
		for _, vm := range containerValue.VolumeMounts {
			if _, ok := buildStrategyVolumesMap[vm.Name]; !ok {
				return nil, fmt.Errorf("volume for the Volume Mount %q is not found", vm.Name)
			}
			volumeMounts[vm.Name] = vm.ReadOnly
		}
	}

	return volumeMounts, nil
}

// generateBaseTaskSpecParams creates the base parameters for TaskSpec
func generateBaseTaskSpecParams() []pipelineapi.ParamSpec {
	return []pipelineapi.ParamSpec{
		{
			Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputImage),
			Description: "The URL of the image that the build produces",
			Type:        pipelineapi.ParamTypeString,
		},
		{
			Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputInsecure),
			Description: "A flag indicating that the output image is on an insecure container registry",
			Type:        pipelineapi.ParamTypeString,
		},
		{
			Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramSourceContext),
			Description: "The context directory inside the source directory",
			Type:        pipelineapi.ParamTypeString,
		},
		{
			Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramSourceRoot),
			Description: "The source directory",
			Type:        pipelineapi.ParamTypeString,
		},
	}
}

// generateBaseTaskSpecWorkspaces creates the base workspaces for TaskSpec
func generateBaseTaskSpecWorkspaces() []pipelineapi.WorkspaceDeclaration {
	return []pipelineapi.WorkspaceDeclaration{
		{
			Name: workspaceSource,
		},
	}
}

// generateBaseTaskSpecResults creates the base results for TaskSpec
func generateBaseTaskSpecResults() []pipelineapi.TaskResult {
	return append(getTaskSpecResults(), getFailureDetailsTaskSpecResults()...)
}

// generateTaskSpecVolumes creates and appends volumes to TaskSpec
func generateTaskSpecVolumes(
	taskSpec *pipelineapi.TaskSpec,
	volumeMounts map[string]bool,
	buildStrategyVolumes []buildv1beta1.BuildStrategyVolume,
	buildVolumes []buildv1beta1.BuildVolume,
	buildRunVolumes []buildv1beta1.BuildVolume,
) error {
	volumes, err := volumes.TaskSpecVolumes(volumeMounts, buildStrategyVolumes, buildVolumes, buildRunVolumes)
	if err != nil {
		return fmt.Errorf("failed to create TaskSpec volumes from volume configurations: %w", err)
	}
	taskSpec.Volumes = append(taskSpec.Volumes, volumes...)
	return nil
}

// generateTaskRunLabels creates labels for the TaskRun
func generateTaskRunLabels(build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) map[string]string {
	taskRunLabels := map[string]string{
		buildv1beta1.LabelBuildRun:           buildRun.Name,
		buildv1beta1.LabelBuildRunGeneration: strconv.FormatInt(buildRun.Generation, 10),
	}

	// Add Build name reference unless it is an embedded Build (empty build name)
	if build.Name != "" {
		taskRunLabels[buildv1beta1.LabelBuild] = build.Name
		taskRunLabels[buildv1beta1.LabelBuildGeneration] = strconv.FormatInt(build.Generation, 10)
	}

	return taskRunLabels
}

// applySourcesToTaskSpec amends TaskSpec with source-related configuration
func applySourcesToTaskSpec(cfg *config.Config, taskSpec *pipelineapi.TaskSpec, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) {
	AmendTaskSpecWithSources(cfg, taskSpec, build, buildRun)
}

// addStrategyParametersToTaskSpec adds strategy-defined parameters to TaskSpec
func addStrategyParametersToTaskSpec(taskSpec *pipelineapi.TaskSpec, parameterDefinitions []buildv1beta1.Parameter) {
	for _, parameterDefinition := range parameterDefinitions {
		param := pipelineapi.ParamSpec{
			Name:        parameterDefinition.Name,
			Description: parameterDefinition.Description,
		}

		switch parameterDefinition.Type {
		case "": // string is default
			fallthrough
		case buildv1beta1.ParameterTypeString:
			param.Type = pipelineapi.ParamTypeString
			if parameterDefinition.Default != nil {
				param.Default = &pipelineapi.ParamValue{
					Type:      pipelineapi.ParamTypeString,
					StringVal: *parameterDefinition.Default,
				}
			}

		case buildv1beta1.ParameterTypeArray:
			param.Type = pipelineapi.ParamTypeArray
			if parameterDefinition.Defaults != nil {
				param.Default = &pipelineapi.ParamValue{
					Type:     pipelineapi.ParamTypeArray,
					ArrayVal: *parameterDefinition.Defaults,
				}
			}
		}

		taskSpec.Params = append(taskSpec.Params, param)
	}
}

// applyOutputImageSteps configures image processing steps
func applyOutputImageSteps(cfg *config.Config, taskRun *pipelineapi.TaskRun, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) error {
	buildRunOutput := buildRun.Spec.Output
	if buildRunOutput == nil {
		buildRunOutput = &buildv1beta1.Image{}
	}

	// Setup image processing, this can be a no-op if no annotations or labels need to be mutated,
	// and if the strategy is pushing the image by not using $(params.shp-output-directory)
	if err := SetupImageProcessing(taskRun, cfg, buildRun.CreationTimestamp.Time, build.Spec.Output, *buildRunOutput); err != nil {
		return fmt.Errorf("failed to setup image processing for BuildRun %q: %w",
			buildRun.Name, err)
	}

	return nil
}

// applyNodeSelectors configures node selectors for the TaskRun
func applyNodeSelectors(taskRun *pipelineapi.TaskRun, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) error {
	taskRunPodTemplate := &pod.PodTemplate{}
	if taskRun.Spec.PodTemplate != nil {
		taskRunPodTemplate = taskRun.Spec.PodTemplate
	}

	// Merge Build and BuildRun NodeSelectors, giving preference to BuildRun NodeSelector
	taskRunNodeSelector := mergeMaps(build.Spec.NodeSelector, buildRun.Spec.NodeSelector)
	if len(taskRunNodeSelector) > 0 {
		taskRunPodTemplate.NodeSelector = taskRunNodeSelector
		taskRun.Spec.PodTemplate = taskRunPodTemplate
	}

	return nil
}

// applyTolerations configures tolerations for the TaskRun
func applyTolerations(taskRun *pipelineapi.TaskRun, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) error {
	taskRunPodTemplate := &pod.PodTemplate{}
	if taskRun.Spec.PodTemplate != nil {
		taskRunPodTemplate = taskRun.Spec.PodTemplate
	}

	// Merge Build and BuildRun Tolerations, giving preference to BuildRun Tolerations values
	taskRunTolerations := mergeTolerations(build.Spec.Tolerations, buildRun.Spec.Tolerations)
	if len(taskRunTolerations) > 0 {
		for i, toleration := range taskRunTolerations {
			if toleration.Effect == "" {
				// set unspecified effects to TaintEffectNoSchedule, as that is the only supported effect
				taskRunTolerations[i].Effect = corev1.TaintEffectNoSchedule
			}
		}
		taskRunPodTemplate.Tolerations = taskRunTolerations
		taskRun.Spec.PodTemplate = taskRunPodTemplate
	}

	return nil
}

// applyScheduler configures scheduler for the TaskRun
func applyScheduler(taskRun *pipelineapi.TaskRun, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) error {
	taskRunPodTemplate := &pod.PodTemplate{}
	if taskRun.Spec.PodTemplate != nil {
		taskRunPodTemplate = taskRun.Spec.PodTemplate
	}

	// Set custom scheduler name if specified, giving preference to BuildRun values
	if buildRun.Spec.SchedulerName != nil {
		taskRunPodTemplate.SchedulerName = *buildRun.Spec.SchedulerName
		taskRun.Spec.PodTemplate = taskRunPodTemplate
	} else if build.Spec.SchedulerName != nil {
		taskRunPodTemplate.SchedulerName = *build.Spec.SchedulerName
		taskRun.Spec.PodTemplate = taskRunPodTemplate
	}

	return nil
}

// applyRuntimeClassName configures runtimeClassName for the TaskRun
func applyRuntimeClassName(taskRun *pipelineapi.TaskRun, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) error {
	taskRunPodTemplate := &pod.PodTemplate{}
	if taskRun.Spec.PodTemplate != nil {
		taskRunPodTemplate = taskRun.Spec.PodTemplate
	}

	// Set runtimeClassName if specified, giving preference to BuildRun values
	if buildRun.Spec.RuntimeClassName != nil {
		taskRunPodTemplate.RuntimeClassName = buildRun.Spec.RuntimeClassName
		taskRun.Spec.PodTemplate = taskRunPodTemplate
	} else if build.Spec.RuntimeClassName != nil {
		taskRunPodTemplate.RuntimeClassName = build.Spec.RuntimeClassName
		taskRun.Spec.PodTemplate = taskRunPodTemplate
	}

	return nil
}

// applyAnnotationsAndLabels configures annotations and labels for the TaskRun
func applyAnnotationsAndLabels(taskRun *pipelineapi.TaskRun, strategy buildv1beta1.BuilderStrategy) error {
	// assign the annotations from the build strategy, filter out those that should not be propagated
	taskRunAnnotations := make(map[string]string)
	for key, value := range strategy.GetAnnotations() {
		if isPropagatableAnnotation(key) {
			taskRunAnnotations[key] = value
		}
	}

	// Update the security context of the Shipwright-injected steps with the runAs user of the build strategy
	steps.UpdateSecurityContext(taskRun.Spec.TaskSpec, taskRunAnnotations, strategy.GetBuildSteps(), strategy.GetSecurityContext())

	if len(taskRunAnnotations) > 0 {
		if taskRun.Annotations == nil {
			taskRun.Annotations = make(map[string]string)
		}
		for k, v := range taskRunAnnotations {
			taskRun.Annotations[k] = v
		}
	}

	// Apply resource labels from strategy
	if taskRun.Labels == nil {
		taskRun.Labels = make(map[string]string)
	}
	for label, value := range strategy.GetResourceLabels() {
		taskRun.Labels[label] = value
	}

	return nil
}

// applyParameters configures parameters for the TaskRun
func applyParameters(taskRun *pipelineapi.TaskRun, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun, strategy buildv1beta1.BuilderStrategy) error {
	// retrieve expected imageURL from build or buildRun
	var image string
	if buildRun.Spec.Output != nil {
		image = buildRun.Spec.Output.Image
	} else {
		image = build.Spec.Output.Image
	}

	insecure := false
	if buildRun.Spec.Output != nil && buildRun.Spec.Output.Insecure != nil {
		insecure = *buildRun.Spec.Output.Insecure
	} else if build.Spec.Output.Insecure != nil {
		insecure = *build.Spec.Output.Insecure
	}

	params := []pipelineapi.Param{
		{
			// shp-output-image
			Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputImage),
			Value: pipelineapi.ParamValue{
				Type:      pipelineapi.ParamTypeString,
				StringVal: image,
			},
		},
		{
			// shp-output-insecure
			Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputInsecure),
			Value: pipelineapi.ParamValue{
				Type:      pipelineapi.ParamTypeString,
				StringVal: strconv.FormatBool(insecure),
			},
		},
		{
			Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramSourceRoot),
			Value: pipelineapi.ParamValue{
				Type:      pipelineapi.ParamTypeString,
				StringVal: "/workspace/source",
			},
		},
	}

	if build.Spec.Source != nil && build.Spec.Source.ContextDir != nil {
		params = append(params, pipelineapi.Param{
			Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramSourceContext),
			Value: pipelineapi.ParamValue{
				Type:      pipelineapi.ParamTypeString,
				StringVal: path.Join("/workspace/source", *build.Spec.Source.ContextDir),
			},
		})
	} else {
		params = append(params, pipelineapi.Param{
			Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramSourceContext),
			Value: pipelineapi.ParamValue{
				Type:      pipelineapi.ParamTypeString,
				StringVal: "/workspace/source",
			},
		})
	}

	taskRun.Spec.Params = params

	// Ensure a proper override of params between Build and BuildRun
	// A BuildRun can override a param as long as it was defined in the Build
	paramValues := OverrideParams(build.Spec.ParamValues, buildRun.Spec.ParamValues)

	// Append params to the TaskRun spec definition
	for _, paramValue := range paramValues {
		parameterDefinition := FindParameterByName(strategy.GetParameters(), paramValue.Name)
		if parameterDefinition == nil {
			// this error should never happen because we validate this upfront in ValidateBuildRunParameters
			return fmt.Errorf("the parameter %q is not defined in the build strategy %q", paramValue.Name, strategy.GetName())
		}

		if err := HandleTaskRunParam(taskRun, parameterDefinition, paramValue); err != nil {
			return err
		}
	}

	return nil
}

// applyTimeout configures timeout for the TaskRun
func applyTimeout(taskRun *pipelineapi.TaskRun, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) error {
	taskRun.Spec.Timeout = effectiveTimeout(build, buildRun)
	return nil
}

func effectiveTimeout(build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) *metav1.Duration {
	if buildRun.Spec.Timeout != nil {
		return buildRun.Spec.Timeout

	} else if build.Spec.Timeout != nil {
		return build.Spec.Timeout
	}

	return nil
}

// mergeEnvironmentVariables combines Build and BuildRun environment variables
func mergeEnvironmentVariables(build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) ([]corev1.EnvVar, error) {
	return env.MergeEnvVars(buildRun.Spec.Env, build.Spec.Env, true)
}

// mergeTolerations merges the values for Spec.Tolerations in the given Build and BuildRun objects, with values in the BuildRun object overriding values
// in the Build object (if present).
func mergeTolerations(buildTolerations []corev1.Toleration, buildRunTolerations []corev1.Toleration) []corev1.Toleration {
	mergedTolerations := []corev1.Toleration{}
	mergedTolerations = append(mergedTolerations, buildRunTolerations...)
	for _, toleration := range buildTolerations {
		if !slices.ContainsFunc(mergedTolerations, func(t corev1.Toleration) bool {
			return t.Key == toleration.Key
		}) {
			mergedTolerations = append(mergedTolerations, toleration)
		}
	}
	return mergedTolerations
}

// isPropagatableAnnotation filters the last-applied-configuration annotation from kubectl because this would break the meaning of this annotation on the target object;
// also, annotations using our own custom resource domains are filtered out because we have no annotations with a semantic for both TaskRun and Pod
func isPropagatableAnnotation(key string) bool {
	return key != "kubectl.kubernetes.io/last-applied-configuration" &&
		!strings.HasPrefix(key, buildv1beta1.ClusterBuildStrategyDomain+"/") &&
		!strings.HasPrefix(key, buildv1beta1.BuildStrategyDomain+"/") &&
		!strings.HasPrefix(key, buildv1beta1.BuildDomain+"/") &&
		!strings.HasPrefix(key, buildv1beta1.BuildRunDomain+"/")
}

func toVolumeMap(strategyVolumes []buildv1beta1.BuildStrategyVolume) map[string]bool {
	res := make(map[string]bool)
	for _, v := range strategyVolumes {
		res[v.Name] = true
	}
	return res
}
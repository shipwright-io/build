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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/env"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/steps"
	"github.com/shipwright-io/build/pkg/volumes"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func createBaseTaskSpec() *pipelineapi.TaskSpec {
	return &pipelineapi.TaskSpec{
		Params:     generateBaseTaskSpecParams(),
		Workspaces: generateBaseTaskSpecWorkspaces(),
		Results:    generateBaseTaskSpecResults(),
	}
}

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

// generateBaseTaskParamReferences creates parameter references using $(params.xxx) syntax.
func generateBaseTaskParamReferences() []pipelineapi.Param {
	return []pipelineapi.Param{
		{Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputImage), Value: pipelineapi.ParamValue{Type: pipelineapi.ParamTypeString, StringVal: fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramOutputImage)}},
		{Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputInsecure), Value: pipelineapi.ParamValue{Type: pipelineapi.ParamTypeString, StringVal: fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramOutputInsecure)}},
		{Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramSourceRoot), Value: pipelineapi.ParamValue{Type: pipelineapi.ParamTypeString, StringVal: fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramSourceRoot)}},
		{Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramSourceContext), Value: pipelineapi.ParamValue{Type: pipelineapi.ParamTypeString, StringVal: fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramSourceContext)}},
	}
}

func generateBaseParamValues(build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) []pipelineapi.Param {
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
			Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputImage),
			Value: pipelineapi.ParamValue{
				Type:      pipelineapi.ParamTypeString,
				StringVal: image,
			},
		},
		{
			Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputInsecure),
			Value: pipelineapi.ParamValue{
				Type:      pipelineapi.ParamTypeString,
				StringVal: fmt.Sprintf("%t", insecure),
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
				StringVal: fmt.Sprintf("/workspace/source/%s", *build.Spec.Source.ContextDir),
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

	return params
}

func generateBaseTaskSpecWorkspaces() []pipelineapi.WorkspaceDeclaration {
	return []pipelineapi.WorkspaceDeclaration{
		{
			Name: workspaceSource,
		},
	}
}

func generateBaseTaskSpecResults() []pipelineapi.TaskResult {
	return append(getTaskSpecResults(), getFailureDetailsTaskSpecResults()...)
}

func applySourcesToTaskSpec(cfg *config.Config, taskSpec *pipelineapi.TaskSpec, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) {
	AmendTaskSpecWithSources(cfg, taskSpec, build, buildRun)
}

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

func applyBuildStrategySteps(
	taskSpec *pipelineapi.TaskSpec,
	build *buildv1beta1.Build,
	buildRun *buildv1beta1.BuildRun,
	buildSteps []buildv1beta1.Step,
	buildStrategyVolumes []buildv1beta1.BuildStrategyVolume,
	combinedEnvs []corev1.EnvVar,
) (map[string]bool, error) {
	volumeMounts := make(map[string]bool)
	buildStrategyVolumesMap := toVolumeMap(buildStrategyVolumes)

	// Build step resource overrides map (step name -> resources)
	stepResourceOverrides := buildStepResourceOverridesMap(build, buildRun)

	for _, containerValue := range buildSteps {
		stepEnv, err := env.MergeEnvVars(combinedEnvs, containerValue.Env, false)
		if err != nil {
			return nil, fmt.Errorf("error(s) occurred merging environment variables into BuildStrategy %q steps: %s", build.Spec.StrategyName(), err.Error())
		}

		// Use overridden resources if specified, otherwise use strategy defaults
		stepResources := containerValue.Resources
		if override, exists := stepResourceOverrides[containerValue.Name]; exists {
			stepResources = override
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
			ComputeResources: stepResources,
			Env:              stepEnv,
		}

		taskSpec.Steps = append(taskSpec.Steps, step)

		for _, vm := range containerValue.VolumeMounts {
			if _, ok := buildStrategyVolumesMap[vm.Name]; !ok {
				return nil, fmt.Errorf("volume for the Volume Mount %q is not found", vm.Name)
			}
			volumeMounts[vm.Name] = vm.ReadOnly
		}
	}

	return volumeMounts, nil
}

// buildStepResourceOverridesMap creates a map of step name to resource requirements
// by merging Build and BuildRun step resource overrides with the following precedence:
// 1. BuildRun.Spec.StepResources (highest priority - for name reference builds)
// 2. BuildRun.Spec.Build.Spec.Strategy.StepResources (for embedded spec builds)
// 3. Build.Spec.Strategy.StepResources (Build-level overrides)
func buildStepResourceOverridesMap(build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) map[string]corev1.ResourceRequirements {
	overrides := make(map[string]corev1.ResourceRequirements)

	// First apply Build-level overrides (lowest priority)
	if build != nil {
		for _, sr := range build.Spec.Strategy.StepResources {
			overrides[sr.Name] = sr.Resources
		}
	}

	// Then apply BuildRun.Spec.Build.Spec.Strategy.StepResources (for embedded spec path)
	if buildRun != nil && buildRun.Spec.Build.Spec != nil {
		for _, sr := range buildRun.Spec.Build.Spec.Strategy.StepResources {
			overrides[sr.Name] = sr.Resources
		}
	}

	// Finally apply BuildRun.Spec.StepResources (highest priority - for name reference path)
	if buildRun != nil {
		for _, sr := range buildRun.Spec.StepResources {
			overrides[sr.Name] = sr.Resources
		}
	}

	return overrides
}

func generateTaskSpecVolumes(
	taskSpec *pipelineapi.TaskSpec,
	volumeMounts map[string]bool,
	buildStrategyVolumes []buildv1beta1.BuildStrategyVolume,
	buildVolumes []buildv1beta1.BuildVolume,
	buildRunVolumes []buildv1beta1.BuildVolume,
) error {
	volumes, err := volumes.TaskSpecVolumes(volumeMounts, buildStrategyVolumes, buildVolumes, buildRunVolumes)
	if err != nil {
		return err
	}
	taskSpec.Volumes = append(taskSpec.Volumes, volumes...)
	return nil
}

func generateTaskRunMetadata(build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		GenerateName: buildRun.Name + "-",
		Namespace:    buildRun.Namespace,
		Labels:       generateTaskRunLabels(build, buildRun),
	}
}

func generateTaskRunLabels(build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) map[string]string {
	taskRunLabels := map[string]string{
		buildv1beta1.LabelBuildRun:           buildRun.Name,
		buildv1beta1.LabelBuildRunGeneration: strconv.FormatInt(buildRun.Generation, 10),
	}

	if build.Name != "" {
		taskRunLabels[buildv1beta1.LabelBuild] = build.Name
		taskRunLabels[buildv1beta1.LabelBuildGeneration] = strconv.FormatInt(build.Generation, 10)
	}

	return taskRunLabels
}

func generateWorkspaceBindings() []pipelineapi.WorkspaceBinding {
	return []pipelineapi.WorkspaceBinding{
		{
			Name:     workspaceSource,
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

func (g *PipelineRunGenerator) applySecurityContextToTaskSpec(taskSpec *pipelineapi.TaskSpec) {
	taskSpecAnnotations := make(map[string]string)
	steps.UpdateSecurityContext(taskSpec, taskSpecAnnotations, g.strategy.GetBuildSteps(), g.strategy.GetSecurityContext())

	if len(taskSpecAnnotations) > 0 {
		if g.pipelineRun.Annotations == nil {
			g.pipelineRun.Annotations = make(map[string]string)
		}
		for k, v := range taskSpecAnnotations {
			g.pipelineRun.Annotations[k] = v
		}
	}
}

func applyNodeSelectors(taskRun *pipelineapi.TaskRun, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) error {
	taskRunPodTemplate := &pod.PodTemplate{}
	if taskRun.Spec.PodTemplate != nil {
		taskRunPodTemplate = taskRun.Spec.PodTemplate
	}

	taskRunNodeSelector := mergeMaps(build.Spec.NodeSelector, buildRun.Spec.NodeSelector)
	if len(taskRunNodeSelector) > 0 {
		taskRunPodTemplate.NodeSelector = taskRunNodeSelector
		taskRun.Spec.PodTemplate = taskRunPodTemplate
	}

	return nil
}

func applyTolerations(taskRun *pipelineapi.TaskRun, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) error {
	taskRunPodTemplate := &pod.PodTemplate{}
	if taskRun.Spec.PodTemplate != nil {
		taskRunPodTemplate = taskRun.Spec.PodTemplate
	}

	taskRunTolerations := mergeTolerations(build.Spec.Tolerations, buildRun.Spec.Tolerations)
	if len(taskRunTolerations) > 0 {
		for i, toleration := range taskRunTolerations {
			if toleration.Effect == "" {
				taskRunTolerations[i].Effect = corev1.TaintEffectNoSchedule
			}
		}
		taskRunPodTemplate.Tolerations = taskRunTolerations
		taskRun.Spec.PodTemplate = taskRunPodTemplate
	}

	return nil
}

func applyScheduler(taskRun *pipelineapi.TaskRun, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) error {
	taskRunPodTemplate := &pod.PodTemplate{}
	if taskRun.Spec.PodTemplate != nil {
		taskRunPodTemplate = taskRun.Spec.PodTemplate
	}

	if buildRun.Spec.SchedulerName != nil {
		taskRunPodTemplate.SchedulerName = *buildRun.Spec.SchedulerName
		taskRun.Spec.PodTemplate = taskRunPodTemplate
	} else if build.Spec.SchedulerName != nil {
		taskRunPodTemplate.SchedulerName = *build.Spec.SchedulerName
		taskRun.Spec.PodTemplate = taskRunPodTemplate
	}

	return nil
}

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

func applyAnnotationsAndLabels(taskRun *pipelineapi.TaskRun, strategy buildv1beta1.BuilderStrategy) error {
	taskRunAnnotations := make(map[string]string)
	for key, value := range strategy.GetAnnotations() {
		if isPropagatableAnnotation(key) {
			taskRunAnnotations[key] = value
		}
	}

	steps.UpdateSecurityContext(taskRun.Spec.TaskSpec, taskRunAnnotations, strategy.GetBuildSteps(), strategy.GetSecurityContext())

	if len(taskRunAnnotations) > 0 {
		if taskRun.Annotations == nil {
			taskRun.Annotations = make(map[string]string)
		}
		for k, v := range taskRunAnnotations {
			taskRun.Annotations[k] = v
		}
	}

	if taskRun.Labels == nil {
		taskRun.Labels = make(map[string]string)
	}
	for label, value := range strategy.GetResourceLabels() {
		taskRun.Labels[label] = value
	}

	return nil
}

func applyParameters(taskRun *pipelineapi.TaskRun, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun, strategy buildv1beta1.BuilderStrategy) error {
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
			Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputImage),
			Value: pipelineapi.ParamValue{
				Type:      pipelineapi.ParamTypeString,
				StringVal: image,
			},
		},
		{
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

	taskRun.Spec.Params = append(taskRun.Spec.Params, params...)

	paramValues := OverrideParams(build.Spec.ParamValues, buildRun.Spec.ParamValues)

	for _, paramValue := range paramValues {
		parameterDefinition := FindParameterByName(strategy.GetParameters(), paramValue.Name)
		if parameterDefinition == nil {
			return fmt.Errorf("the parameter %q is not defined in the build strategy %q", paramValue.Name, strategy.GetName())
		}

		if err := HandleTaskRunParam(taskRun, parameterDefinition, paramValue); err != nil {
			return err
		}
	}

	return nil
}

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

// PipelineRun-specific helpers

func createBasePipelineSpec() *pipelineapi.PipelineSpec {
	return &pipelineapi.PipelineSpec{
		Tasks:      []pipelineapi.PipelineTask{},
		Params:     generateBaseTaskSpecParams(),
		Workspaces: generateBasePipelineWorkspaceDeclarations(),
	}
}

func generateBasePipelineWorkspaceDeclarations() []pipelineapi.PipelineWorkspaceDeclaration {
	return []pipelineapi.PipelineWorkspaceDeclaration{
		{
			Name:        workspaceSource,
			Description: "Workspace for source code and build artifacts",
		},
		{
			Name:        "cache",
			Description: "Cache workspace for build artifacts",
			Optional:    true,
		},
	}
}

func generatePipelineTaskRunTemplate(serviceAccountName string) pipelineapi.PipelineTaskRunTemplate {
	return pipelineapi.PipelineTaskRunTemplate{
		ServiceAccountName: serviceAccountName,
	}
}

func generatePipelineWorkspaceBindings() []pipelineapi.WorkspaceBinding {
	return []pipelineapi.WorkspaceBinding{
		{
			Name: workspaceSource,
			VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			},
		},
		{
			Name:     "cache",
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

func createSourceAcquisitionPipelineTask(taskSpec *pipelineapi.TaskSpec) pipelineapi.PipelineTask {
	return pipelineapi.PipelineTask{
		Name: "source-acquisition",
		TaskSpec: &pipelineapi.EmbeddedTask{
			TaskSpec: *taskSpec,
		},
		Params:     generateBaseTaskParamReferences(),
		Workspaces: generateSourceAcquisitionWorkspaceBindings(),
	}
}

func generateSourceAcquisitionWorkspaceBindings() []pipelineapi.WorkspacePipelineTaskBinding {
	return []pipelineapi.WorkspacePipelineTaskBinding{
		{
			Name:      workspaceSource,
			Workspace: workspaceSource,
		},
	}
}

func addStrategyParametersToPipelineSpec(pipelineSpec *pipelineapi.PipelineSpec, parameterDefinitions []buildv1beta1.Parameter) {
	for _, parameterDefinition := range parameterDefinitions {
		param := pipelineapi.ParamSpec{
			Name:        parameterDefinition.Name,
			Description: parameterDefinition.Description,
		}

		switch parameterDefinition.Type {
		case "", buildv1beta1.ParameterTypeString:
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

		pipelineSpec.Params = append(pipelineSpec.Params, param)
	}
}

func createBuildStrategyPipelineTask(taskSpec *pipelineapi.TaskSpec, strategy buildv1beta1.BuilderStrategy) pipelineapi.PipelineTask {
	pipelineTask := pipelineapi.PipelineTask{
		Name: "build-strategy",
		TaskSpec: &pipelineapi.EmbeddedTask{
			TaskSpec: *taskSpec,
		},
		Params:     generateBuildStrategyTaskParams(strategy),
		Workspaces: generateBuildStrategyWorkspaceBindings(),
		RunAfter:   []string{"source-acquisition"},
	}

	// Add shp-output-directory parameter if it exists in the TaskSpec
	if hasTaskSpecParam(taskSpec, "shp-output-directory") {
		pipelineTask.Params = append(pipelineTask.Params, pipelineapi.Param{
			Name: "shp-output-directory",
			Value: pipelineapi.ParamValue{
				Type:      pipelineapi.ParamTypeString,
				StringVal: "$(params.shp-output-directory)",
			},
		})
	}

	return pipelineTask
}

func generateBuildStrategyTaskParams(strategy buildv1beta1.BuilderStrategy) []pipelineapi.Param {
	params := generateBaseTaskParamReferences()

	for _, strategyParam := range strategy.GetParameters() {
		var paramRef string
		if strategyParam.Type == buildv1beta1.ParameterTypeArray {
			paramRef = fmt.Sprintf("$(params.%s[*])", strategyParam.Name)
		} else {
			paramRef = fmt.Sprintf("$(params.%s)", strategyParam.Name)
		}

		params = append(params, pipelineapi.Param{
			Name: strategyParam.Name,
			Value: pipelineapi.ParamValue{
				Type:      pipelineapi.ParamTypeString,
				StringVal: paramRef,
			},
		})
	}

	return params
}

func generateBuildStrategyWorkspaceBindings() []pipelineapi.WorkspacePipelineTaskBinding {
	return []pipelineapi.WorkspacePipelineTaskBinding{
		{
			Name:      workspaceSource,
			Workspace: workspaceSource,
		},
		{
			Name:      "cache",
			Workspace: "cache",
		},
	}
}

func addOutputDirectoryParamToPipelineSpec(pipelineSpec *pipelineapi.PipelineSpec) {
	prefixedOutputDirectory := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputDirectory)
	pipelineSpec.Params = append(pipelineSpec.Params, pipelineapi.ParamSpec{
		Name: prefixedOutputDirectory,
		Type: pipelineapi.ParamTypeString,
	})
}

func createOutputImagePipelineTask(taskSpec *pipelineapi.TaskSpec, hasOutputDirectory bool) pipelineapi.PipelineTask {
	task := pipelineapi.PipelineTask{
		Name: "output-image",
		TaskSpec: &pipelineapi.EmbeddedTask{
			TaskSpec: *taskSpec,
		},
		Params: generateBaseTaskParamReferences(),
		Workspaces: []pipelineapi.WorkspacePipelineTaskBinding{
			{Name: workspaceSource, Workspace: workspaceSource},
		},
		RunAfter: []string{"build-strategy"},
	}

	if hasOutputDirectory {
		prefixedOutputDirectory := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputDirectory)
		task.Params = append(task.Params, pipelineapi.Param{
			Name: prefixedOutputDirectory,
			Value: pipelineapi.ParamValue{
				Type:      pipelineapi.ParamTypeString,
				StringVal: fmt.Sprintf("$(params.%s)", prefixedOutputDirectory),
			},
		})
	}

	return task
}

func hasOutputDirectoryParam(pipelineSpec *pipelineapi.PipelineSpec) bool {
	prefixedOutputDirectory := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputDirectory)
	for _, param := range pipelineSpec.Params {
		if param.Name == prefixedOutputDirectory {
			return true
		}
	}
	return false
}

func hasTaskSpecParam(taskSpec *pipelineapi.TaskSpec, paramName string) bool {
	for _, param := range taskSpec.Params {
		if param.Name == paramName {
			return true
		}
	}
	return false
}

func doesTaskSpecReferenceOutputDirectory(taskSpec *pipelineapi.TaskSpec) bool {
	prefixedOutputDirectory := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputDirectory)
	for _, step := range taskSpec.Steps {
		if isStepReferencingParameter(&step, prefixedOutputDirectory) {
			return true
		}
	}
	return false
}

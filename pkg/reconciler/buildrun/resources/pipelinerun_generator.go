// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PipelineRunGenerator implements BuildRunExecutorGenerator for PipelineRun execution.
//
// For single-arch builds, each build phase runs as a separate Task:
//   - source-acquisition
//   - build-strategy
//   - output-image
//
// For multi-arch builds (when spec.output.multiArch is set), the pipeline fans out:
//   - source-acquisition (clone + push source as OCI artifact)
//   - build-<os>-<arch> per platform (parallel, each with per-task nodeSelector)
//   - assemble-index (creates OCI image index from per-platform results)
type PipelineRunGenerator struct {
	cfg                *config.Config
	build              *buildv1beta1.Build
	buildRun           *buildv1beta1.BuildRun
	serviceAccountName string
	strategy           buildv1beta1.BuilderStrategy

	pipelineRun   *pipelineapi.PipelineRun
	pipelineTasks []pipelineapi.PipelineTask
}

func NewPipelineRunGenerator(
	cfg *config.Config,
	build *buildv1beta1.Build,
	buildRun *buildv1beta1.BuildRun,
	serviceAccountName string,
	strategy buildv1beta1.BuilderStrategy,
) *PipelineRunGenerator {
	return &PipelineRunGenerator{
		cfg:                cfg,
		build:              build,
		buildRun:           buildRun,
		serviceAccountName: serviceAccountName,
		strategy:           strategy,
		pipelineTasks:      []pipelineapi.PipelineTask{},
	}
}

func (g *PipelineRunGenerator) isMultiArch() bool {
	ma := g.effectiveMultiArch()
	return ma != nil && len(ma.Platforms) > 0
}

func (g *PipelineRunGenerator) effectiveMultiArch() *buildv1beta1.MultiArch {
	if g.buildRun.Spec.Output != nil && g.buildRun.Spec.Output.MultiArch != nil {
		return g.buildRun.Spec.Output.MultiArch
	}
	return g.build.Spec.Output.MultiArch
}

func (g *PipelineRunGenerator) InitializeExecutor() error {
	pipelineSpec := createBasePipelineSpec()

	var workspaces []pipelineapi.WorkspaceBinding
	if g.isMultiArch() {
		workspaces = generateMultiArchWorkspaceBindings()
	} else {
		workspaces = generatePipelineWorkspaceBindings()
	}

	g.pipelineRun = &pipelineapi.PipelineRun{
		ObjectMeta: generateTaskRunMetadata(g.build, g.buildRun),
		Spec: pipelineapi.PipelineRunSpec{
			PipelineSpec:    pipelineSpec,
			TaskRunTemplate: generatePipelineTaskRunTemplate(g.serviceAccountName),
			Workspaces:      workspaces,
		},
	}

	return nil
}

func (g *PipelineRunGenerator) GenerateSourceAcquisitionPhase(_ *executionContext) error {
	taskSpec := createBaseTaskSpec()
	applySourcesToTaskSpec(g.cfg, taskSpec, g.build, g.buildRun)

	if g.isMultiArch() {
		pushStep := generateSourceBundlePushStep(g.cfg, taskSpec, g.build.Spec.Output.PushSecret)
		taskSpec.Steps = append(taskSpec.Steps, pushStep)
	}

	g.applySecurityContextToTaskSpec(taskSpec)

	pipelineTask := createSourceAcquisitionPipelineTask(taskSpec)
	g.pipelineTasks = append(g.pipelineTasks, pipelineTask)

	return nil
}

func (g *PipelineRunGenerator) GenerateBuildStrategyPhase(execCtx *executionContext) error {
	if g.isMultiArch() {
		return g.generateMultiArchBuildPhase(execCtx)
	}
	return g.generateSingleArchBuildPhase(execCtx)
}

func (g *PipelineRunGenerator) generateSingleArchBuildPhase(execCtx *executionContext) error {
	taskSpec := createBaseTaskSpec()
	addStrategyParametersToTaskSpec(taskSpec, g.strategy.GetParameters())

	volumeMounts, err := applyBuildStrategySteps(
		taskSpec,
		g.build,
		g.buildRun,
		g.strategy.GetBuildSteps(),
		g.strategy.GetVolumes(),
		execCtx.combinedEnvs,
	)
	if err != nil {
		return err
	}

	execCtx.volumeMounts = volumeMounts

	if err := generateTaskSpecVolumes(
		taskSpec,
		execCtx.volumeMounts,
		execCtx.strategyVolumes,
		execCtx.buildVolumes,
		execCtx.buildRunVolumes,
	); err != nil {
		return err
	}

	execCtx.hasOutputDirectory = doesTaskSpecReferenceOutputDirectory(taskSpec)

	if execCtx.hasOutputDirectory {
		prefixedOutputDirectory := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputDirectory)
		taskSpec.Params = append(taskSpec.Params, pipelineapi.ParamSpec{
			Name: prefixedOutputDirectory,
			Type: pipelineapi.ParamTypeString,
		})
		addOutputDirectoryParamToPipelineSpec(g.pipelineRun.Spec.PipelineSpec)
	}

	addStrategyParametersToPipelineSpec(g.pipelineRun.Spec.PipelineSpec, g.strategy.GetParameters())
	g.applySecurityContextToTaskSpec(taskSpec)

	pipelineTask := createBuildStrategyPipelineTask(taskSpec, g.strategy)
	g.pipelineTasks = append(g.pipelineTasks, pipelineTask)

	return nil
}

func (g *PipelineRunGenerator) generateMultiArchBuildPhase(execCtx *executionContext) error {
	multiArch := g.effectiveMultiArch()

	addStrategyParametersToPipelineSpec(g.pipelineRun.Spec.PipelineSpec, g.strategy.GetParameters())

	for _, platform := range multiArch.Platforms {
		pipelineTask, err := createPerPlatformBuildTask(
			g.cfg,
			g.build,
			g.buildRun,
			g.strategy,
			platform,
			execCtx,
		)
		if err != nil {
			return fmt.Errorf("generating build task for %s/%s: %w", platform.OS, platform.Arch, err)
		}
		g.applySecurityContextToTaskSpec(&pipelineTask.TaskSpec.TaskSpec)
		g.pipelineTasks = append(g.pipelineTasks, pipelineTask)
	}

	if execCtx.hasOutputDirectory {
		addOutputDirectoryParamToPipelineSpec(g.pipelineRun.Spec.PipelineSpec)
	}

	return nil
}

func (g *PipelineRunGenerator) GenerateOutputImagePhase(execCtx *executionContext) error {
	if g.isMultiArch() {
		return g.generateMultiArchOutputPhase()
	}
	return g.generateSingleArchOutputPhase(execCtx)
}

func (g *PipelineRunGenerator) generateSingleArchOutputPhase(execCtx *executionContext) error {
	buildRunOutput := g.buildRun.Spec.Output
	if buildRunOutput == nil {
		buildRunOutput = &buildv1beta1.Image{}
	}

	hasSourceTimestamp := true
	stepArgs, err := BuildImageProcessingArgs(
		g.cfg,
		g.buildRun.CreationTimestamp.Time,
		g.build.Spec.Output,
		*buildRunOutput,
		execCtx.hasOutputDirectory,
		hasSourceTimestamp,
	)
	if err != nil {
		return err
	}

	if len(stepArgs) == 0 {
		return nil
	}

	taskSpec := createBaseTaskSpec()

	if execCtx.hasOutputDirectory {
		prefixedOutputDirectory := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputDirectory)
		taskSpec.Params = append(taskSpec.Params, pipelineapi.ParamSpec{
			Name: prefixedOutputDirectory,
			Type: pipelineapi.ParamTypeString,
		})
	}

	// Don't add EmptyDir volume - use workspace PVC
	if err := CreateImageProcessingStep(
		g.cfg,
		taskSpec,
		stepArgs,
		false,
		g.build.Spec.Output.PushSecret,
	); err != nil {
		return err
	}

	g.applySecurityContextToTaskSpec(taskSpec)

	pipelineTask := createOutputImagePipelineTask(taskSpec, execCtx.hasOutputDirectory)
	g.pipelineTasks = append(g.pipelineTasks, pipelineTask)

	return nil
}

func (g *PipelineRunGenerator) generateMultiArchOutputPhase() error {
	multiArch := g.effectiveMultiArch()
	assemblyTask := createIndexAssemblyTask(g.cfg, multiArch.Platforms, g.build)
	g.applySecurityContextToTaskSpec(&assemblyTask.TaskSpec.TaskSpec)
	g.pipelineTasks = append(g.pipelineTasks, assemblyTask)
	return nil
}

func (g *PipelineRunGenerator) ApplyInfrastructureConfiguration() error {
	nodeSelector := MergeMaps(g.build.Spec.NodeSelector, g.buildRun.Spec.NodeSelector)

	tolerations := mergeTolerations(g.build.Spec.Tolerations, g.buildRun.Spec.Tolerations)
	for i := range tolerations {
		if tolerations[i].Effect == "" {
			tolerations[i].Effect = corev1.TaintEffectNoSchedule
		}
	}

	var schedulerName string
	if g.buildRun.Spec.SchedulerName != nil {
		schedulerName = *g.buildRun.Spec.SchedulerName
	} else if g.build.Spec.SchedulerName != nil {
		schedulerName = *g.build.Spec.SchedulerName
	}

	var runtimeClassName *string
	if g.buildRun.Spec.RuntimeClassName != nil {
		runtimeClassName = g.buildRun.Spec.RuntimeClassName
	} else if g.build.Spec.RuntimeClassName != nil {
		runtimeClassName = g.build.Spec.RuntimeClassName
	}

	if len(nodeSelector) > 0 || len(tolerations) > 0 || schedulerName != "" || runtimeClassName != nil {
		if g.pipelineRun.Spec.TaskRunTemplate.PodTemplate == nil {
			g.pipelineRun.Spec.TaskRunTemplate.PodTemplate = &pod.PodTemplate{}
		}

		if len(nodeSelector) > 0 {
			g.pipelineRun.Spec.TaskRunTemplate.PodTemplate.NodeSelector = nodeSelector
		}
		if len(tolerations) > 0 {
			g.pipelineRun.Spec.TaskRunTemplate.PodTemplate.Tolerations = tolerations
		}
		if schedulerName != "" {
			g.pipelineRun.Spec.TaskRunTemplate.PodTemplate.SchedulerName = schedulerName
		}
		if runtimeClassName != nil {
			g.pipelineRun.Spec.TaskRunTemplate.PodTemplate.RuntimeClassName = runtimeClassName
		}
	}

	if g.isMultiArch() {
		multiArch := g.effectiveMultiArch()
		g.pipelineRun.Spec.TaskRunSpecs = generateMultiArchTaskRunSpecs(
			multiArch.Platforms,
			nodeSelector,
			tolerations,
		)
	}

	return nil
}

func (g *PipelineRunGenerator) ApplyMetadataConfiguration() error {
	pipelineRunAnnotations := make(map[string]string)
	for key, value := range g.strategy.GetAnnotations() {
		if isPropagatableAnnotation(key) {
			pipelineRunAnnotations[key] = value
		}
	}

	if len(pipelineRunAnnotations) > 0 {
		if g.pipelineRun.Annotations == nil {
			g.pipelineRun.Annotations = make(map[string]string)
		}
		for k, v := range pipelineRunAnnotations {
			g.pipelineRun.Annotations[k] = v
		}
	}

	if g.pipelineRun.Labels == nil {
		g.pipelineRun.Labels = make(map[string]string)
	}
	for label, value := range g.strategy.GetResourceLabels() {
		g.pipelineRun.Labels[label] = value
	}

	g.pipelineRun.Spec.Timeouts = &pipelineapi.TimeoutFields{
		Pipeline: effectiveTimeout(g.build, g.buildRun),
	}

	params := generateBaseParamValues(g.build, g.buildRun)
	paramValues := OverrideParams(g.build.Spec.ParamValues, g.buildRun.Spec.ParamValues)

	for _, paramValue := range paramValues {
		parameterDefinition := FindParameterByName(g.strategy.GetParameters(), paramValue.Name)
		if parameterDefinition == nil {
			return fmt.Errorf("the parameter %q is not defined in the build strategy %q", paramValue.Name, g.strategy.GetName())
		}

		switch parameterDefinition.Type {
		case "", buildv1beta1.ParameterTypeString:
			if paramValue.SingleValue == nil {
				continue
			}

			if paramValue.Value != nil {
				params = append(params, pipelineapi.Param{
					Name: paramValue.Name,
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: *paramValue.Value,
					},
				})
			}

		case buildv1beta1.ParameterTypeArray:
			if paramValue.Values == nil {
				continue
			}

			var arrayValues []string
			for _, v := range paramValue.Values {
				if v.Value != nil {
					arrayValues = append(arrayValues, *v.Value)
				}
			}

			if len(arrayValues) > 0 {
				params = append(params, pipelineapi.Param{
					Name: paramValue.Name,
					Value: pipelineapi.ParamValue{
						Type:     pipelineapi.ParamTypeArray,
						ArrayVal: arrayValues,
					},
				})
			}
		}
	}

	g.pipelineRun.Spec.Params = params

	if hasOutputDirectoryParam(g.pipelineRun.Spec.PipelineSpec) {
		prefixedOutputDirectory := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputDirectory)
		g.pipelineRun.Spec.Params = append(g.pipelineRun.Spec.Params, pipelineapi.Param{
			Name: prefixedOutputDirectory,
			Value: pipelineapi.ParamValue{
				Type:      pipelineapi.ParamTypeString,
				StringVal: "/workspace/source/output-image",
			},
		})
	}

	g.pipelineRun.Spec.PipelineSpec.Tasks = g.pipelineTasks

	return nil
}

func (g *PipelineRunGenerator) GetExecutor() client.Object {
	return g.pipelineRun
}

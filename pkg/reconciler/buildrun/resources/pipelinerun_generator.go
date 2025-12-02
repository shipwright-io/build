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
// Execution Model:
//
//	PipelineRun
//	  ├─ Task 1: source-acquisition
//	  │    └─ Step: git-clone
//	  ├─ Task 2: build-strategy
//	  │    └─ Step: buildah-build
//	  └─ Task 3: output-image
//	       └─ Step: image-push
//
// Each phase runs as a separate Task within the Pipeline.
// Tasks communicate via workspace volumes (PersistentVolumeClaim templates).
//
// Future: Can be extended to support multi-arch builds with parallel build tasks:
//
//	├─ Task 2a: build-amd64
//	├─ Task 2b: build-arm64 (runs in parallel with 2a)
//	└─ Task 3: manifest-push (waits for 2a and 2b)
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

func (g *PipelineRunGenerator) InitializeExecutor() error {
	pipelineSpec := createBasePipelineSpec()

	g.pipelineRun = &pipelineapi.PipelineRun{
		ObjectMeta: generateTaskRunMetadata(g.build, g.buildRun),
		Spec: pipelineapi.PipelineRunSpec{
			PipelineSpec:    pipelineSpec,
			TaskRunTemplate: generatePipelineTaskRunTemplate(g.serviceAccountName),
			Workspaces:      generatePipelineWorkspaceBindings(),
		},
	}

	return nil
}

func (g *PipelineRunGenerator) GenerateSourceAcquisitionPhase(execCtx *executionContext) error {
	taskSpec := createBaseTaskSpec()
	applySourcesToTaskSpec(g.cfg, taskSpec, g.build, g.buildRun)

	pipelineTask := createSourceAcquisitionPipelineTask(taskSpec)
	g.pipelineTasks = append(g.pipelineTasks, pipelineTask)

	return nil
}

func (g *PipelineRunGenerator) GenerateBuildStrategyPhase(execCtx *executionContext) error {
	// taskSpec := createBaseTaskSpec()

	// addStrategyParametersToTaskSpec(taskSpec, g.strategy.GetParameters())

	// volumeMounts, err := applyBuildStrategySteps(
	// 	taskSpec,
	// 	g.build,
	// 	g.strategy.GetBuildSteps(),
	// 	g.strategy.GetVolumes(),
	// 	execCtx.combinedEnvs,
	// )
	// if err != nil {
	// 	return fmt.Errorf("failed to apply build strategy steps: %w", err)
	// }

	// execCtx.volumeMounts = volumeMounts

	// if err := generateTaskSpecVolumes(
	// 	taskSpec,
	// 	execCtx.volumeMounts,
	// 	execCtx.strategyVolumes,
	// 	execCtx.buildVolumes,
	// 	execCtx.buildRunVolumes,
	// ); err != nil {
	// 	return fmt.Errorf("failed to generate TaskSpec volumes: %w", err)
	// }

	// // Add cache workspace declaration to TaskSpec if strategy references it
	// taskSpec.Workspaces = append(taskSpec.Workspaces, pipelineapi.WorkspaceDeclaration{
	// 	Name:     "cache",
	// 	Optional: true,
	// })

	// // Build params list including base shipwright params and strategy params
	// pipelineTaskParams := generateBaseTaskParamReferences() // Start with base params from helper

	// // Add strategy parameters to Pipeline-level and task-level
	// for _, strategyParam := range g.strategy.GetParameters() {
	// 	// Determine parameter type
	// 	paramType := pipelineapi.ParamTypeString
	// 	if strategyParam.Type == buildv1beta1.ParameterTypeArray {
	// 		paramType = pipelineapi.ParamTypeArray
	// 	}

	// 	// Create Pipeline-level param spec with defaults
	// 	pipelineParamSpec := pipelineapi.ParamSpec{
	// 		Name: strategyParam.Name,
	// 		Type: paramType,
	// 	}

	// 	// Add default value if strategy has one
	// 	if paramType == pipelineapi.ParamTypeString && strategyParam.Default != nil {
	// 		pipelineParamSpec.Default = &pipelineapi.ParamValue{
	// 			Type:      pipelineapi.ParamTypeString,
	// 			StringVal: *strategyParam.Default,
	// 		}
	// 	} else if paramType == pipelineapi.ParamTypeArray && strategyParam.Defaults != nil {
	// 		pipelineParamSpec.Default = &pipelineapi.ParamValue{
	// 			Type:     pipelineapi.ParamTypeArray,
	// 			ArrayVal: *strategyParam.Defaults,
	// 		}
	// 	}

	// 	g.pipelineRun.Spec.PipelineSpec.Params = append(g.pipelineRun.Spec.PipelineSpec.Params, pipelineParamSpec)

	// 	// Add to PipelineTask params (pass through from pipeline params)
	// 	paramRef := fmt.Sprintf("$(params.%s)", strategyParam.Name)
	// 	if paramType == pipelineapi.ParamTypeArray {
	// 		paramRef = fmt.Sprintf("$(params.%s[*])", strategyParam.Name)
	// 	}

	// 	pipelineTaskParams = append(pipelineTaskParams, pipelineapi.Param{
	// 		Name: strategyParam.Name,
	// 		Value: pipelineapi.ParamValue{
	// 			Type:      pipelineapi.ParamTypeString,
	// 			StringVal: paramRef,
	// 		},
	// 	})
	// }

	// pipelineTask := pipelineapi.PipelineTask{
	// 	Name: "build-strategy",
	// 	TaskSpec: &pipelineapi.EmbeddedTask{
	// 		TaskSpec: *taskSpec,
	// 	},
	// 	Params: pipelineTaskParams,
	// 	Workspaces: []pipelineapi.WorkspacePipelineTaskBinding{
	// 		{Name: workspaceSource, Workspace: workspaceSource},
	// 		{Name: "cache", Workspace: "cache"},
	// 	},
	// 	RunAfter: []string{"source-acquisition"},
	// }

	// g.pipelineTasks = append(g.pipelineTasks, pipelineTask)
	return nil
}

func (g *PipelineRunGenerator) GenerateOutputImagePhase(execCtx *executionContext) error {
	// taskSpec := createBaseTaskSpec()

	// buildRunOutput := g.buildRun.Spec.Output
	// if buildRunOutput == nil {
	// 	buildRunOutput = &buildv1beta1.Image{}
	// }

	// tempTaskRun := &pipelineapi.TaskRun{
	// 	Spec: pipelineapi.TaskRunSpec{
	// 		TaskSpec: taskSpec,
	// 	},
	// }

	// if err := SetupImageProcessing(tempTaskRun, g.cfg, g.buildRun.CreationTimestamp.Time, g.build.Spec.Output, *buildRunOutput); err != nil {
	// 	return fmt.Errorf("failed to setup image processing: %w", err)
	// }

	// taskSpec = tempTaskRun.Spec.TaskSpec

	// pipelineTask := pipelineapi.PipelineTask{
	// 	Name: "output-image",
	// 	TaskSpec: &pipelineapi.EmbeddedTask{
	// 		TaskSpec: *taskSpec,
	// 	},
	// 	Workspaces: []pipelineapi.WorkspacePipelineTaskBinding{
	// 		{Name: workspaceSource, Workspace: workspaceSource},
	// 	},
	// 	RunAfter: []string{"build-strategy"},
	// }

	// g.pipelineTasks = append(g.pipelineTasks, pipelineTask)
	return nil
}

func (g *PipelineRunGenerator) ApplyInfrastructureConfiguration() error {
	nodeSelector := mergeMaps(g.build.Spec.NodeSelector, g.buildRun.Spec.NodeSelector)

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

	if len(nodeSelector) > 0 || len(tolerations) > 0 || schedulerName != "" {
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

	// Generate base parameter values using helper from resource_builders.go
	params := generateBaseParamValues(g.build, g.buildRun)

	// Add strategy parameter values (only for explicitly provided params, like TaskRun does)
	paramValues := OverrideParams(g.build.Spec.ParamValues, g.buildRun.Spec.ParamValues)

	for _, paramValue := range paramValues {
		parameterDefinition := FindParameterByName(g.strategy.GetParameters(), paramValue.Name)
		if parameterDefinition == nil {
			return fmt.Errorf("the parameter %q is not defined in the build strategy %q", paramValue.Name, g.strategy.GetName())
		}

		// Handle different parameter types
		switch parameterDefinition.Type {
		case "", buildv1beta1.ParameterTypeString:
			// Handle string parameters
			if paramValue.SingleValue == nil {
				// No value provided, will use default from TaskSpec
				continue
			}

			if paramValue.SingleValue.Value != nil {
				params = append(params, pipelineapi.Param{
					Name: paramValue.Name,
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: *paramValue.SingleValue.Value,
					},
				})
			}
			// TODO: Handle ConfigMapValue and SecretValue if needed

		case buildv1beta1.ParameterTypeArray:
			// Handle array parameters
			if paramValue.Values == nil {
				// No values provided, will use defaults from TaskSpec
				continue
			}

			var arrayValues []string
			for _, v := range paramValue.Values {
				if v.Value != nil {
					arrayValues = append(arrayValues, *v.Value)
				}
				// TODO: Handle ConfigMapValue and SecretValue if needed
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

	g.pipelineRun.Spec.PipelineSpec.Tasks = g.pipelineTasks

	return nil
}

func (g *PipelineRunGenerator) GetExecutor() client.Object {
	return g.pipelineRun
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	g.pipelineRun = &pipelineapi.PipelineRun{
		ObjectMeta: generatePipelineRunMetadata(g.build, g.buildRun),
		Spec: pipelineapi.PipelineRunSpec{
			PipelineSpec: &pipelineapi.PipelineSpec{
				Tasks: []pipelineapi.PipelineTask{},
				Workspaces: []pipelineapi.PipelineWorkspaceDeclaration{
					{Name: workspaceSource, Description: "Workspace for source code and build artifacts"},
				},
			},
			TaskRunTemplate: pipelineapi.PipelineTaskRunTemplate{
				ServiceAccountName: g.serviceAccountName,
			},
			Workspaces: []pipelineapi.WorkspaceBinding{
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
			},
		},
	}

	return nil
}

func (g *PipelineRunGenerator) GenerateSourceAcquisitionPhase(execCtx *executionContext) error {
	taskSpec := createBaseTaskSpec()
	applySourcesToTaskSpec(g.cfg, taskSpec, g.build, g.buildRun)

	pipelineTask := pipelineapi.PipelineTask{
		Name: "source-acquisition",
		TaskSpec: &pipelineapi.EmbeddedTask{
			TaskSpec: *taskSpec,
		},
		Workspaces: []pipelineapi.WorkspacePipelineTaskBinding{
			{Name: workspaceSource, Workspace: workspaceSource},
		},
	}

	g.pipelineTasks = append(g.pipelineTasks, pipelineTask)
	return nil
}

func (g *PipelineRunGenerator) GenerateBuildStrategyPhase(execCtx *executionContext) error {
	taskSpec := createBaseTaskSpec()

	addStrategyParametersToTaskSpec(taskSpec, g.strategy.GetParameters())

	volumeMounts, err := applyBuildStrategySteps(
		taskSpec,
		g.build,
		g.strategy.GetBuildSteps(),
		g.strategy.GetVolumes(),
		execCtx.combinedEnvs,
	)
	if err != nil {
		return fmt.Errorf("failed to apply build strategy steps: %w", err)
	}

	execCtx.volumeMounts = volumeMounts

	if err := generateTaskSpecVolumes(
		taskSpec,
		execCtx.volumeMounts,
		execCtx.strategyVolumes,
		execCtx.buildVolumes,
		execCtx.buildRunVolumes,
	); err != nil {
		return fmt.Errorf("failed to generate TaskSpec volumes: %w", err)
	}

	pipelineTask := pipelineapi.PipelineTask{
		Name: "build-strategy",
		TaskSpec: &pipelineapi.EmbeddedTask{
			TaskSpec: *taskSpec,
		},
		Workspaces: []pipelineapi.WorkspacePipelineTaskBinding{
			{Name: workspaceSource, Workspace: workspaceSource},
		},
		RunAfter: []string{"source-acquisition"},
	}

	g.pipelineTasks = append(g.pipelineTasks, pipelineTask)
	return nil
}

func (g *PipelineRunGenerator) GenerateOutputImagePhase(execCtx *executionContext) error {
	taskSpec := createBaseTaskSpec()

	buildRunOutput := g.buildRun.Spec.Output
	if buildRunOutput == nil {
		buildRunOutput = &buildv1beta1.Image{}
	}

	tempTaskRun := &pipelineapi.TaskRun{
		Spec: pipelineapi.TaskRunSpec{
			TaskSpec: taskSpec,
		},
	}

	if err := SetupImageProcessing(tempTaskRun, g.cfg, g.buildRun.CreationTimestamp.Time, g.build.Spec.Output, *buildRunOutput); err != nil {
		return fmt.Errorf("failed to setup image processing: %w", err)
	}

	taskSpec = tempTaskRun.Spec.TaskSpec

	pipelineTask := pipelineapi.PipelineTask{
		Name: "output-image",
		TaskSpec: &pipelineapi.EmbeddedTask{
			TaskSpec: *taskSpec,
		},
		Workspaces: []pipelineapi.WorkspacePipelineTaskBinding{
			{Name: workspaceSource, Workspace: workspaceSource},
		},
		RunAfter: []string{"build-strategy"},
	}

	g.pipelineTasks = append(g.pipelineTasks, pipelineTask)
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

	var image string
	if g.buildRun.Spec.Output != nil {
		image = g.buildRun.Spec.Output.Image
	} else {
		image = g.build.Spec.Output.Image
	}

	insecure := false
	if g.buildRun.Spec.Output != nil && g.buildRun.Spec.Output.Insecure != nil {
		insecure = *g.buildRun.Spec.Output.Insecure
	} else if g.build.Spec.Output.Insecure != nil {
		insecure = *g.build.Spec.Output.Insecure
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

	if g.build.Spec.Source != nil && g.build.Spec.Source.ContextDir != nil {
		params = append(params, pipelineapi.Param{
			Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramSourceContext),
			Value: pipelineapi.ParamValue{
				Type:      pipelineapi.ParamTypeString,
				StringVal: fmt.Sprintf("/workspace/source/%s", *g.build.Spec.Source.ContextDir),
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

	g.pipelineRun.Spec.Params = params

	g.pipelineRun.Spec.PipelineSpec.Tasks = g.pipelineTasks

	return nil
}

func (g *PipelineRunGenerator) GetExecutor() client.Object {
	return g.pipelineRun
}

func generatePipelineRunMetadata(build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		GenerateName: buildRun.Name + "-",
		Namespace:    buildRun.Namespace,
		Labels:       generateTaskRunLabels(build, buildRun),
	}
}

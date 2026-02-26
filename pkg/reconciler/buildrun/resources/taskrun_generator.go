// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TaskRunGenerator implements BuildRunExecutorGenerator for TaskRun execution.
//
// All build phases run as sequential steps within a single TaskSpec:
//   - Source acquisition (git-clone)
//   - Build strategy execution (buildah-build)
//   - Output processing (image-push)
//
// Steps share the same Pod filesystem for efficient data transfer.
type TaskRunGenerator struct {
	cfg                *config.Config
	build              *buildv1beta1.Build
	buildRun           *buildv1beta1.BuildRun
	serviceAccountName string
	strategy           buildv1beta1.BuilderStrategy

	taskRun *pipelineapi.TaskRun
}

func NewTaskRunGenerator(
	cfg *config.Config,
	build *buildv1beta1.Build,
	buildRun *buildv1beta1.BuildRun,
	serviceAccountName string,
	strategy buildv1beta1.BuilderStrategy,
) *TaskRunGenerator {
	return &TaskRunGenerator{
		cfg:                cfg,
		build:              build,
		buildRun:           buildRun,
		serviceAccountName: serviceAccountName,
		strategy:           strategy,
	}
}

func (g *TaskRunGenerator) InitializeExecutor() error {
	taskSpec := createBaseTaskSpec()

	g.taskRun = &pipelineapi.TaskRun{
		ObjectMeta: generateTaskRunMetadata(g.build, g.buildRun),
		Spec: pipelineapi.TaskRunSpec{
			ServiceAccountName: g.serviceAccountName,
			TaskSpec:           taskSpec,
			Workspaces:         generateWorkspaceBindings(),
		},
	}

	return nil
}

func (g *TaskRunGenerator) GenerateSourceAcquisitionPhase(_ *executionContext) error {
	applySourcesToTaskSpec(g.cfg, g.taskRun.Spec.TaskSpec, g.build, g.buildRun)
	return nil
}

func (g *TaskRunGenerator) GenerateBuildStrategyPhase(execCtx *executionContext) error {
	addStrategyParametersToTaskSpec(g.taskRun.Spec.TaskSpec, g.strategy.GetParameters())

	volumeMounts, err := applyBuildStrategySteps(
		g.taskRun.Spec.TaskSpec,
		g.build,
		g.strategy.GetBuildSteps(),
		g.strategy.GetVolumes(),
		execCtx.combinedEnvs,
	)
	if err != nil {
		return err
	}

	execCtx.volumeMounts = volumeMounts

	if err = generateTaskSpecVolumes(
		g.taskRun.Spec.TaskSpec,
		execCtx.volumeMounts,
		execCtx.strategyVolumes,
		execCtx.buildVolumes,
		execCtx.buildRunVolumes,
	); err != nil {
		return err
	}

	return nil
}

func (g *TaskRunGenerator) GenerateOutputImagePhase(_ *executionContext) error {
	buildRunOutput := g.buildRun.Spec.Output
	if buildRunOutput == nil {
		buildRunOutput = &buildv1beta1.Image{}
	}

	if err := SetupImageProcessing(g.taskRun, g.cfg, g.buildRun.CreationTimestamp.Time, g.build.Spec.Output, *buildRunOutput); err != nil {
		return err
	}

	return nil
}

func (g *TaskRunGenerator) ApplyInfrastructureConfiguration() error {
	if err := applyNodeSelectors(g.taskRun, g.build, g.buildRun); err != nil {
		return err
	}

	if err := applyTolerations(g.taskRun, g.build, g.buildRun); err != nil {
		return err
	}

	if err := applyRuntimeClassName(g.taskRun, g.build, g.buildRun); err != nil {
		return err
	}

	if err := applyScheduler(g.taskRun, g.build, g.buildRun); err != nil {
		return err
	}

	if err := addCertificates(g.taskRun, g.build, g.buildRun); err != nil {
		return err
	}

	return nil
}

func (g *TaskRunGenerator) ApplyMetadataConfiguration() error {
	if err := applyAnnotationsAndLabels(g.taskRun, g.strategy); err != nil {
		return err
	}

	if err := applyTimeout(g.taskRun, g.build, g.buildRun); err != nil {
		return err
	}

	if err := applyParameters(g.taskRun, g.build, g.buildRun, g.strategy); err != nil {
		return err
	}

	return nil
}

func (g *TaskRunGenerator) GetExecutor() client.Object {
	return g.taskRun
}

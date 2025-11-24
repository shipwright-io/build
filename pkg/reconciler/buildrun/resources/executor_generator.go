// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/env"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// executionContext holds pre-computed, executor-agnostic data needed for build execution.
type executionContext struct {
	combinedEnvs    []corev1.EnvVar
	volumeMounts    map[string]bool
	strategyVolumes []buildv1beta1.BuildStrategyVolume
	buildVolumes    []buildv1beta1.BuildVolume
	buildRunVolumes []buildv1beta1.BuildVolume
}

func prepareExecutionContext(
	build *buildv1beta1.Build,
	buildRun *buildv1beta1.BuildRun,
	strategy buildv1beta1.BuilderStrategy,
) (*executionContext, error) {
	combinedEnvs, err := mergeEnvironmentVariables(build, buildRun)
	if err != nil {
		return nil, fmt.Errorf("failed to merge environment variables: %w", err)
	}

	return &executionContext{
		combinedEnvs:    combinedEnvs,
		volumeMounts:    make(map[string]bool),
		strategyVolumes: strategy.GetVolumes(),
		buildVolumes:    build.Spec.Volumes,
		buildRunVolumes: buildRun.Spec.Volumes,
	}, nil
}

func mergeEnvironmentVariables(build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) ([]corev1.EnvVar, error) {
	return env.MergeEnvVars(buildRun.Spec.Env, build.Spec.Env, true)
}

// BuildRunExecutorGenerator defines the interface for generating build execution objects (TaskRun or PipelineRun).
type BuildRunExecutorGenerator interface {
	InitializeExecutor() error
	GenerateSourceAcquisitionPhase(execCtx *executionContext) error
	GenerateBuildStrategyPhase(execCtx *executionContext) error
	GenerateOutputImagePhase(execCtx *executionContext) error
	ApplyInfrastructureConfiguration() error
	ApplyMetadataConfiguration() error
	GetExecutor() client.Object
}

// GenerateBuildRunExecutor orchestrates the build execution generation using the Template Method pattern.
func GenerateBuildRunExecutor(
	cfg *config.Config,
	build *buildv1beta1.Build,
	buildRun *buildv1beta1.BuildRun,
	serviceAccountName string,
	strategy buildv1beta1.BuilderStrategy,
	generator BuildRunExecutorGenerator,
) (client.Object, error) {
	execCtx, err := prepareExecutionContext(build, buildRun, strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare execution context for BuildRun %q: %w", buildRun.Name, err)
	}

	if err := generator.InitializeExecutor(); err != nil {
		return nil, fmt.Errorf("failed to initialize executor for BuildRun %q: %w", buildRun.Name, err)
	}

	if err := generator.GenerateSourceAcquisitionPhase(execCtx); err != nil {
		return nil, fmt.Errorf("phase 1 (source acquisition) failed for BuildRun %q: %w", buildRun.Name, err)
	}

	if err := generator.GenerateBuildStrategyPhase(execCtx); err != nil {
		return nil, fmt.Errorf("phase 2 (build strategy execution) failed for BuildRun %q: %w", buildRun.Name, err)
	}

	if err := generator.GenerateOutputImagePhase(execCtx); err != nil {
		return nil, fmt.Errorf("phase 3 (output image processing) failed for BuildRun %q: %w", buildRun.Name, err)
	}

	if err := generator.ApplyInfrastructureConfiguration(); err != nil {
		return nil, fmt.Errorf("phase 4 (infrastructure configuration) failed for BuildRun %q: %w", buildRun.Name, err)
	}

	if err := generator.ApplyMetadataConfiguration(); err != nil {
		return nil, fmt.Errorf("phase 5 (metadata and runtime configuration) failed for BuildRun %q: %w", buildRun.Name, err)
	}

	return generator.GetExecutor(), nil
}

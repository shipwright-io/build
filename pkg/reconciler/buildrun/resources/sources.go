// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/sources"

	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const defaultSourceName = "default"

// isLocalCopyBuildSource appends all "Sources" in a single slice, and if any entry is typed
// "LocalCopy" it returns first LocalCopy typed BuildSource found, or nil.
func isLocalCopyBuildSource(
	build *buildv1alpha1.Build,
	buildRun *buildv1alpha1.BuildRun,
) *buildv1alpha1.BuildSource {
	sources := []buildv1alpha1.BuildSource{}
	if build.Spec.Sources != nil {
		sources = append(sources, build.Spec.Sources...)
	}
	if buildRun.Spec.Sources != nil {
		sources = append(sources, *buildRun.Spec.Sources...)
	}
	for _, source := range sources {
		if source.Type == buildv1alpha1.LocalCopy {
			return &source
		}
	}
	return nil
}

// AmendTaskSpecWithSources adds the necessary steps to either wait for user upload ("LocalCopy"), or
// alternatively, configures the Task steps to use bundle and "git clone".
func AmendTaskSpecWithSources(
	cfg *config.Config,
	taskSpec *pipeline.TaskSpec,
	build *buildv1alpha1.Build,
	buildRun *buildv1alpha1.BuildRun,
) {
	if localCopy := isLocalCopyBuildSource(build, buildRun); localCopy != nil {
		sources.AppendLocalCopyStep(cfg, taskSpec, localCopy.Timeout)
	} else {
		// create the step for spec.source, either Git or Bundle
		switch {
		case build.Spec.Source.BundleContainer != nil:
			sources.AppendBundleStep(cfg, taskSpec, build.Spec.Source, defaultSourceName)
		case build.Spec.Source.URL != nil:
			sources.AppendGitStep(cfg, taskSpec, build.Spec.Source, defaultSourceName)
		}
	}

	// inspecting .spec.sources looking for "http" typed sources to generate the TaskSpec items
	// in order to handle remote artifacts
	for _, source := range build.Spec.Sources {
		if source.Type == buildv1alpha1.HTTP {
			sources.AppendHTTPStep(cfg, taskSpec, source)
		}
	}
}

func updateBuildRunStatusWithSourceResult(buildrun *buildv1alpha1.BuildRun, results []pipeline.TaskRunResult) {
	buildSpec := buildrun.Status.BuildSpec

	switch {
	case buildSpec.Source.BundleContainer != nil:
		sources.AppendBundleResult(buildrun, defaultSourceName, results)

	case buildSpec.Source.URL != nil:
		sources.AppendGitResult(buildrun, defaultSourceName, results)
	}

	// no results for HTTP sources yet
}

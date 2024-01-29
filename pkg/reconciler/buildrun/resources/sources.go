// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/sources"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const defaultSourceName = "default"

const sourceTimestampName = "source-timestamp"

// isLocalCopyBuildSource appends all "Sources" in a single slice, and if any entry is typed
// "LocalCopy" it returns first LocalCopy typed BuildSource found, or nil.
func isLocalCopyBuildSource(
	build *buildv1alpha1.Build,
	buildRun *buildv1alpha1.BuildRun,
) *buildv1alpha1.BuildSource {
	sources := []buildv1alpha1.BuildSource{}

	sources = append(sources, build.Spec.Sources...)
	sources = append(sources, buildRun.Spec.Sources...)

	for _, source := range sources {
		if source.Type == buildv1alpha1.LocalCopy {
			return &source
		}
	}
	return nil
}

func appendSourceTimestampResult(taskSpec *pipelineapi.TaskSpec) {
	taskSpec.Results = append(taskSpec.Results,
		pipelineapi.TaskResult{
			Name:        sources.TaskResultName(defaultSourceName, sourceTimestampName),
			Description: "The timestamp of the source.",
		},
	)
}

// AmendTaskSpecWithSources adds the necessary steps to either wait for user upload ("LocalCopy"), or
// alternatively, configures the Task steps to use bundle and "git clone".
func AmendTaskSpecWithSources(
	cfg *config.Config,
	taskSpec *pipelineapi.TaskSpec,
	build *buildv1alpha1.Build,
	buildRun *buildv1alpha1.BuildRun,
) {
	if localCopy := isLocalCopyBuildSource(build, buildRun); localCopy != nil {
		sources.AppendLocalCopyStep(cfg, taskSpec, localCopy.Timeout)
	} else {
		// create the step for spec.source, either Git or Bundle
		switch {
		case build.Spec.Source.BundleContainer != nil:
			appendSourceTimestampResult(taskSpec)
			sources.AppendBundleStep(cfg, taskSpec, build.Spec.Source, defaultSourceName)
		case build.Spec.Source.URL != nil:
			appendSourceTimestampResult(taskSpec)
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

func updateBuildRunStatusWithSourceResult(buildrun *buildv1alpha1.BuildRun, results []pipelineapi.TaskRunResult) {
	buildSpec := buildrun.Status.BuildSpec

	// no results for HTTP sources yet
	switch {
	case buildSpec.Source.BundleContainer != nil:
		sources.AppendBundleResult(buildrun, defaultSourceName, results)

	case buildSpec.Source.URL != nil:
		sources.AppendGitResult(buildrun, defaultSourceName, results)
	}

	if sourceTimestamp := sources.FindResultValue(results, defaultSourceName, sourceTimestampName); strings.TrimSpace(sourceTimestamp) != "" {
		if sec, err := strconv.ParseInt(sourceTimestamp, 10, 64); err == nil {
			for i := range buildrun.Status.Sources {
				if buildrun.Status.Sources[i].Name == defaultSourceName {
					buildrun.Status.Sources[i].Timestamp = &metav1.Time{Time: time.Unix(sec, 0)}
				}
			}
		}
	}
}

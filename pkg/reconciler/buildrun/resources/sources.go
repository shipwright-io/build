// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/sources"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const defaultSourceName = "default"

const sourceTimestampName = "source-timestamp"

// isLocalCopyBuildSource appends all "Sources" in a single slice, and if any entry is typed
// "LocalCopy" it returns first LocalCopy typed BuildSource found, or nil.
func isLocalCopyBuildSource(
	build *buildv1beta1.Build,
	buildRun *buildv1beta1.BuildRun,
) *buildv1beta1.Local {
	if buildRun.Spec.Source != nil && buildRun.Spec.Source.Type == buildv1beta1.LocalType {
		return buildRun.Spec.Source.Local
	}

	if build.Spec.Source != nil && build.Spec.Source.Type == buildv1beta1.LocalType {
		return build.Spec.Source.Local
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
	build *buildv1beta1.Build,
	buildRun *buildv1beta1.BuildRun,
) {
	if localCopy := isLocalCopyBuildSource(build, buildRun); localCopy != nil {
		sources.AppendLocalCopyStep(cfg, taskSpec, localCopy.Timeout)
	} else if build.Spec.Source != nil {

		// create the step for spec.source, either Git or Bundle
		switch build.Spec.Source.Type {
		case buildv1beta1.OCIArtifactType:
			if build.Spec.Source.OCIArtifact != nil {
				appendSourceTimestampResult(taskSpec)
				sources.AppendBundleStep(cfg, taskSpec, build.Spec.Source.OCIArtifact, defaultSourceName)
			}
		case buildv1beta1.GitType:
			if build.Spec.Source.Git != nil {
				appendSourceTimestampResult(taskSpec)
				sources.AppendGitStep(cfg, taskSpec, *build.Spec.Source.Git, defaultSourceName)
			}
		}
	}
}

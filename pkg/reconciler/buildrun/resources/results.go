// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"

	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const (
	imageDigestResult = "image-digest"
	imageSizeResult   = "image-size"
)

// UpdateBuildRunUsingTaskResults surface the task results
// to the buildrun
func UpdateBuildRunUsingTaskResults(
	buildRun *build.BuildRun,
	taskRunResult []pipeline.TaskRunResult,
) {
	// Set source results
	updateBuildRunStatusWithSourceResult(buildRun, taskRunResult)

	// Initializing output result
	buildRun.Status.Output = &build.Output{}

	// Set output results
	updateBuildRunStatusWithOutputResult(buildRun, taskRunResult)
}

func updateBuildRunStatusWithOutputResult(buildRun *build.BuildRun, taskRunResult []pipeline.TaskRunResult) {
	for _, result := range taskRunResult {
		switch result.Name {
		case generateOutputResultName(imageDigestResult):
			buildRun.Status.Output.Digest = result.Value

		case generateOutputResultName(imageSizeResult):
			buildRun.Status.Output.Size = result.Value
		}
	}
}

func generateOutputResultName(resultName string) string {
	return fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultName)
}

func getTaskSpecResults() []pipeline.TaskResult {
	return []pipeline.TaskResult{
		{
			Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, imageDigestResult),
			Description: "The digest of the image",
		},
		{
			Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, imageSizeResult),
			Description: "The compressed size of the image",
		},
	}
}

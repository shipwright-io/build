// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"fmt"
	"strconv"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/ctxlog"

	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	imageDigestResult = "IMAGE_DIGEST"
	imageSizeResult   = "image-size"
)

// UpdateBuildRunUsingTaskResults surface the task results
// to the buildrun
func UpdateBuildRunUsingTaskResults(
	ctx context.Context,
	buildRun *build.BuildRun,
	taskRunResult []pipeline.TaskRunResult,
	request reconcile.Request,
) {
	// Set source results
	updateBuildRunStatusWithSourceResult(buildRun, taskRunResult)

	// Initializing output result
	buildRun.Status.Output = &build.Output{}

	// Set output results
	updateBuildRunStatusWithOutputResult(ctx, buildRun, taskRunResult, request)
}

func updateBuildRunStatusWithOutputResult(ctx context.Context, buildRun *build.BuildRun, taskRunResult []pipeline.TaskRunResult, request reconcile.Request) {
	for _, result := range taskRunResult {
		switch result.Name {
		case generateOutputResultName(imageDigestResult):
			buildRun.Status.Output.Digest = result.Value

		case generateOutputResultName(imageSizeResult):
			if size, err := strconv.ParseInt(result.Value, 10, 64); err != nil {
				ctxlog.Info(ctx, "invalid value for output image size from taskRun result", namespace, request.Namespace, name, request.Name, "error", err)
			} else {
				buildRun.Status.Output.Size = size
			}
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
		{
			Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputImage),
			Description: "The url of the built image",
		},
	}
}

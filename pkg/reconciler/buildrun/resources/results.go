// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/ctxlog"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	imageDigestResult    = "image-digest"
	imageSizeResult      = "image-size"
	imageVulnerabilities = "image-vulnerabilities"
)

// UpdateBuildRunUsingTaskResults surface the task results
// to the buildrun
func UpdateBuildRunUsingTaskResults(
	ctx context.Context,
	buildRun *buildapi.BuildRun,
	taskRunResult []pipelineapi.TaskRunResult,
	request reconcile.Request,
) {
	// Set source results
	updateBuildRunStatusWithSourceResult(buildRun, taskRunResult)

	// Set output results
	updateBuildRunStatusWithOutputResult(ctx, buildRun, taskRunResult, request)
}

func updateBuildRunStatusWithOutputResult(ctx context.Context, buildRun *buildapi.BuildRun, taskRunResult []pipelineapi.TaskRunResult, request reconcile.Request) {
	if buildRun.Status.Output == nil {
		buildRun.Status.Output = &buildapi.Output{}
	}

	for _, result := range taskRunResult {
		switch result.Name {
		case generateOutputResultName(imageDigestResult):
			buildRun.Status.Output.Digest = result.Value.StringVal

		case generateOutputResultName(imageSizeResult):
			if size, err := strconv.ParseInt(result.Value.StringVal, 10, 64); err != nil {
				ctxlog.Info(ctx, "invalid value for output image size from taskRun result", namespace, request.Namespace, name, request.Name, "error", err)
			} else {
				buildRun.Status.Output.Size = size
			}
		case generateOutputResultName(imageVulnerabilities):
			if err := json.Unmarshal([]byte(result.Value.StringVal), &buildRun.Status.Output.Vulnerabilities); err != nil {
				ctxlog.Info(ctx, "failed to unmarshal vulnerabilities list", namespace, request.Namespace, name, request.Name, "error", err)
				buildRun.Status.Output.Vulnerabilities = nil
			}
		}
	}
}

func generateOutputResultName(resultName string) string {
	return fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultName)
}

func getTaskSpecResults() []pipelineapi.TaskResult {
	return []pipelineapi.TaskResult{
		{
			Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, imageDigestResult),
			Description: "The digest of the image",
		},
		{
			Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, imageSizeResult),
			Description: "The compressed size of the image",
		},
		{
			Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, imageVulnerabilities),
			Description: "List of vulnerabilities",
		},
	}
}

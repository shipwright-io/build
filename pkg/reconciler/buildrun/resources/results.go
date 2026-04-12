// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/ctxlog"
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
			buildRun.Status.Output.Vulnerabilities = getImageVulnerabilitiesResult(result)
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

func getImageVulnerabilitiesResult(result pipelineapi.TaskRunResult) []buildapi.Vulnerability {
	var vulns []buildapi.Vulnerability
	if len(result.Value.StringVal) == 0 {
		return vulns
	}

	vulnerabilities := strings.Split(result.Value.StringVal, ",")
	for _, vulnerability := range vulnerabilities {
		vuln := strings.Split(vulnerability, ":")
		severity := getSeverity(vuln[1])
		vulns = append(vulns, buildapi.Vulnerability{
			ID:       vuln[0],
			Severity: severity,
		})
	}
	return vulns
}

func getSeverity(sev string) buildapi.VulnerabilitySeverity {
	switch strings.ToUpper(sev) {
	case "L":
		return buildapi.Low
	case "M":
		return buildapi.Medium
	case "H":
		return buildapi.High
	case "C":
		return buildapi.Critical
	default:
		return buildapi.Unknown
	}
}

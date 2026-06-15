// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"k8s.io/apimachinery/pkg/labels"
	"knative.dev/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

// applyAssembleIndexOutputDigest sets status.output.digest to the OCI image index digest
// produced by assemble-index (same convention as ko/buildkit multi-platform: output.digest
// refers to the manifest list when applicable).
func applyAssembleIndexOutputDigest(buildRun *buildapi.BuildRun, digest string) {
	if buildRun.Status.Output == nil {
		buildRun.Status.Output = &buildapi.Output{}
	}
	buildRun.Status.Output.Digest = digest
}

// UpdateBuildRunWithMultiArchResults extracts per-platform build results from a PipelineRun's
// child TaskRuns and populates the BuildRun's platformResults status field; the assemble-index
// task supplies the manifest-list digest via status.output.digest.
// A non-nil error is returned for any TaskRun fetch failure that is not a NotFound
func UpdateBuildRunWithMultiArchResults(
	ctx context.Context,
	buildRun *buildapi.BuildRun,
	pipelineRun *pipelineapi.PipelineRun,
	platforms []buildapi.ImagePlatform,
	c client.Client,
) error {
	if pipelineRun == nil || len(platforms) == 0 {
		return nil
	}

	// Fetch all child TaskRuns in a single List call
	var taskRunList pipelineapi.TaskRunList
	if err := c.List(ctx, &taskRunList, client.InNamespace(pipelineRun.Namespace), client.MatchingLabelsSelector{
		Selector: labels.SelectorFromSet(labels.Set{
			pipeline.PipelineRunLabelKey: pipelineRun.Name,
		}),
	}); err != nil {
		return fmt.Errorf("listing TaskRuns for PipelineRun %s: %w", pipelineRun.Name, err)
	}

	taskRunsByPipelineTask := make(map[string]*pipelineapi.TaskRun, len(taskRunList.Items))
	for i := range taskRunList.Items {
		tr := &taskRunList.Items[i]
		if ptName, ok := tr.Labels[pipeline.PipelineTaskLabelKey]; ok {
			taskRunsByPipelineTask[ptName] = tr
		}
	}

	digestResultName := generateOutputResultName(imageDigestResult)
	sizeResultName := generateOutputResultName(imageSizeResult)
	vulnResultName := generateOutputResultName(imageVulnerabilities)

	buildRun.Status.PlatformResults = make([]buildapi.PlatformBuildResult, 0, len(platforms))

	if buildRun.Status.Output != nil {
		buildRun.Status.Output.Digest = ""
		buildRun.Status.Output.Size = 0
		buildRun.Status.Output.Vulnerabilities = nil
	}

	var allVulnerabilities []buildapi.Vulnerability
	allPlatformsTerminal := true

	for _, p := range platforms {
		taskName := platformTaskName(p)
		result := buildapi.PlatformBuildResult{
			Platform: p,
			Status:   buildapi.PlatformBuildStatusPending,
		}

		taskRun, exists := taskRunsByPipelineTask[taskName]
		if !exists {
			allPlatformsTerminal = false
			buildRun.Status.PlatformResults = append(buildRun.Status.PlatformResults, result)
			continue
		}

		condition := taskRun.Status.GetCondition(apis.ConditionSucceeded)
		switch {
		case condition == nil:
			result.Status = buildapi.PlatformBuildStatusPending
			allPlatformsTerminal = false
		case condition.IsTrue():
			result.Status = buildapi.PlatformBuildStatusSucceeded
		case condition.IsFalse():
			result.Status = buildapi.PlatformBuildStatusFailed
			result.FailureMessage = condition.Message
		default:
			result.Status = buildapi.PlatformBuildStatusRunning
			allPlatformsTerminal = false
		}

		for _, tr := range taskRun.Status.Results {
			switch tr.Name {
			case digestResultName:
				result.Digest = tr.Value.StringVal
			case sizeResultName:
				if size, err := strconv.ParseInt(tr.Value.StringVal, 10, 64); err == nil {
					result.Size = size
				}
			case vulnResultName:
				result.Vulnerabilities = getImageVulnerabilitiesResult(tr)
				allVulnerabilities = append(allVulnerabilities, result.Vulnerabilities...)
			}
		}

		buildRun.Status.PlatformResults = append(buildRun.Status.PlatformResults, result)
	}

	if assembleTaskRun, ok := taskRunsByPipelineTask["assemble-index"]; ok {
		if buildRun.Status.Output == nil {
			buildRun.Status.Output = &buildapi.Output{}
		}
		for _, tr := range assembleTaskRun.Status.Results {
			if tr.Name == digestResultName {
				applyAssembleIndexOutputDigest(buildRun, tr.Value.StringVal)
			}
		}
		buildRun.Status.Output.Size = 0
		if allPlatformsTerminal {
			buildRun.Status.Output.Vulnerabilities = allVulnerabilities
		}
	}

	return nil
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/ctxlog"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	buildRun *build.BuildRun,
	taskRunResult []pipelineapi.TaskRunResult,
	request reconcile.Request,
) {
	// Set source results
	updateBuildRunStatusWithSourceResult(buildRun, taskRunResult)

	// Set output results
	updateBuildRunStatusWithOutputResult(ctx, buildRun, taskRunResult, request)
}

func updateBuildRunStatusWithOutputResult(ctx context.Context, buildRun *build.BuildRun, taskRunResult []pipelineapi.TaskRunResult, request reconcile.Request) {
	if buildRun.Status.Output == nil {
		buildRun.Status.Output = &build.Output{}
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

func getImageVulnerabilitiesResult(result pipelineapi.TaskRunResult) []build.Vulnerability {
	var vulns []build.Vulnerability
	if len(result.Value.StringVal) == 0 {
		return vulns
	}

	vulnerabilities := strings.Split(result.Value.StringVal, ",")
	for _, vulnerability := range vulnerabilities {
		vuln := strings.Split(vulnerability, ":")
		severity := getSeverity(vuln[1])
		vulns = append(vulns, build.Vulnerability{
			ID:       vuln[0],
			Severity: severity,
		})
	}
	return vulns
}

func getSeverity(sev string) build.VulnerabilitySeverity {
	switch strings.ToUpper(sev) {
	case "L":
		return build.Low
	case "M":
		return build.Medium
	case "H":
		return build.High
	case "C":
		return build.Critical
	default:
		return build.Unknown
	}
}

// UpdateBuildRunWithMultiArchResults extracts per-platform build results and the
// manifest digest from a PipelineRun's child TaskRuns and populates the
// BuildRun's PlatformResults and ManifestDigest status fields.
func UpdateBuildRunWithMultiArchResults(
	ctx context.Context,
	buildRun *build.BuildRun,
	pipelineRun *pipelineapi.PipelineRun,
	platforms []build.ImagePlatform,
	c client.Client,
) {
	if pipelineRun == nil || len(platforms) == 0 {
		return
	}

	// Build a map from PipelineTaskName -> ChildStatusReference
	childRefMap := make(map[string]pipelineapi.ChildStatusReference, len(pipelineRun.Status.ChildReferences))
	for _, ref := range pipelineRun.Status.ChildReferences {
		childRefMap[ref.PipelineTaskName] = ref
	}

	digestResultName := generateOutputResultName(imageDigestResult)
	sizeResultName := generateOutputResultName(imageSizeResult)
	vulnResultName := generateOutputResultName(imageVulnerabilities)

	buildRun.Status.PlatformResults = make([]build.PlatformBuildResult, 0, len(platforms))

	var allVulnerabilities []build.Vulnerability

	for _, p := range platforms {
		taskName := platformTaskName(p)
		result := build.PlatformBuildResult{
			Platform: p,
			Status:   build.PlatformBuildStatusPending,
		}

		childRef, exists := childRefMap[taskName]
		if !exists {
			buildRun.Status.PlatformResults = append(buildRun.Status.PlatformResults, result)
			continue
		}

		taskRun := &pipelineapi.TaskRun{}
		if err := c.Get(ctx, types.NamespacedName{
			Namespace: pipelineRun.Namespace,
			Name:      childRef.Name,
		}, taskRun); err != nil {
			result.Status = build.PlatformBuildStatusFailed
			result.FailureMessage = fmt.Sprintf("failed to fetch TaskRun %s: %v", childRef.Name, err)
			buildRun.Status.PlatformResults = append(buildRun.Status.PlatformResults, result)
			continue
		}

		condition := taskRun.Status.GetCondition(apis.ConditionSucceeded)
		switch {
		case condition == nil:
			result.Status = build.PlatformBuildStatusPending
		case condition.IsTrue():
			result.Status = build.PlatformBuildStatusSucceeded
		case condition.IsFalse():
			result.Status = build.PlatformBuildStatusFailed
			result.FailureMessage = condition.Message
		default:
			result.Status = build.PlatformBuildStatusRunning
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

	// Extract ManifestDigest from the assemble-index task and overwrite
	// Output.Digest so the BuildRun reflects the manifest list, not a
	// nondeterministic per-platform digest from the flat result aggregation.
	// Output.Size is cleared because the flat aggregation picks it from an
	// arbitrary per-platform TaskRun; correct per-platform data lives in
	// PlatformResults. Output.Vulnerabilities is set to the union of all
	// per-platform vulnerabilities.
	if assembleRef, ok := childRefMap["assemble-index"]; ok {
		taskRun := &pipelineapi.TaskRun{}
		if err := c.Get(ctx, types.NamespacedName{
			Namespace: pipelineRun.Namespace,
			Name:      assembleRef.Name,
		}, taskRun); err == nil {
			if buildRun.Status.Output == nil {
				buildRun.Status.Output = &build.Output{}
			}
			for _, tr := range taskRun.Status.Results {
				if tr.Name == digestResultName {
					buildRun.Status.ManifestDigest = tr.Value.StringVal
					buildRun.Status.Output.Digest = tr.Value.StringVal
				}
			}
			buildRun.Status.Output.Size = 0
			buildRun.Status.Output.Vulnerabilities = allVulnerabilities
		}
	}
}

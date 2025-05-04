// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const (
	imageDigestResult    = "image-digest"
	imageSizeResult      = "image-size"
	imageVulnerabilities = "image-vulnerabilities"
)

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

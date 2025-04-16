// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const (
	resultErrorMessage         = "error-message"
	resultErrorReason          = "error-reason"
	prefixedResultErrorReason  = prefixParamsResultsVolumes + "-" + resultErrorReason
	prefixedResultErrorMessage = prefixParamsResultsVolumes + "-" + resultErrorMessage
)

func getFailureDetailsTaskSpecResults() []pipelineapi.TaskResult {
	return []pipelineapi.TaskResult{
		{
			Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorMessage),
			Description: "The error description of the task run",
		},
		{
			Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorReason),
			Description: "The error reason of the task run",
		},
	}
}

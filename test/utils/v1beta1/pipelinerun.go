// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package utils

import (
	"fmt"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetPipelineRunFromBuildRun retrieves an owned PipelineRun based on the BuildRunName
func (t *TestBuild) GetPipelineRunFromBuildRun(buildRunName string) (*pipelineapi.PipelineRun, error) {
	pipelineRunLabelSelector := fmt.Sprintf("buildrun.shipwright.io/name=%s", buildRunName)

	prInterface := t.PipelineClientSet.TektonV1().PipelineRuns(t.Namespace)

	prList, err := prInterface.List(t.Context, metav1.ListOptions{
		LabelSelector: pipelineRunLabelSelector,
	})
	if err != nil {
		return nil, err
	}

	if len(prList.Items) == 0 {
		return nil, fmt.Errorf("no PipelineRun found for BuildRun %s/%s with label selector %s", t.Namespace, buildRunName, pipelineRunLabelSelector)
	}

	if len(prList.Items) > 1 {
		return nil, fmt.Errorf("found %d PipelineRuns for BuildRun %s/%s, expected exactly 1. PipelineRuns: %v", len(prList.Items), t.Namespace, buildRunName, prList.Items)
	}

	return &prList.Items[0], nil
}

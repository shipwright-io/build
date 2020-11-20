// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"errors"
	"fmt"

	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/apis"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// This class is intended to host all CRUD calls for testing TaskRuns CRDs resources

// GetTaskRunFromBuildRun retrieves an owned TaskRun based on the BuildRunName
func (t *TestBuild) GetTaskRunFromBuildRun(buildRunName string) (*v1beta1.TaskRun, error) {
	taskRunLabelSelector := fmt.Sprintf("buildrun.build.dev/name=%s", buildRunName)

	trInterface := t.PipelineClientSet.TektonV1beta1().TaskRuns(t.Namespace)

	trList, err := trInterface.List(metav1.ListOptions{
		LabelSelector: taskRunLabelSelector,
	})
	if err != nil {
		return nil, err
	}

	if len(trList.Items) != 1 {
		return nil, fmt.Errorf("failed to find an owned TaskRun based on a Buildrun %s name", buildRunName)
	}

	return &trList.Items[0], nil
}

// UpdateTaskRun applies changes to a provided taskRun object
func (t *TestBuild) UpdateTaskRun(tr *v1beta1.TaskRun) (*v1beta1.TaskRun, error) {
	trInterface := t.PipelineClientSet.TektonV1beta1().TaskRuns(t.Namespace)

	return trInterface.Update(tr)
}

// GetTRReason returns the Reason of the Succeeded condition
// of an existing TaskRun based on a BuildRun name
func (t *TestBuild) GetTRReason(buildRunName string) (string, error) {
	tr, err := t.GetTaskRunFromBuildRun(buildRunName)
	if err != nil {
		return "", err
	}

	trCondition := tr.Status.GetCondition(apis.ConditionSucceeded)
	if trCondition != nil {
		return trCondition.Reason, nil
	}

	return "", errors.New("foo")
}

// GetTRTillDesiredReason polls until a TaskRun matches a desired Reason
// or it exits if an error happen or a timeout is reach.
func (t *TestBuild) GetTRTillDesiredReason(buildRunName string, reason string) (string, error) {
	var trReason string
	var err error

	var (
		pollTRTillCompletion = func() (bool, error) {

			trReason, err = t.GetTRReason(buildRunName)
			if err != nil {
				return false, err
			}

			if trReason == reason {
				return true, nil
			}

			return false, nil
		}
	)

	pollError := wait.PollImmediate(t.Interval, t.TimeOut, pollTRTillCompletion)
	if pollError != nil {
		return trReason, pollError
	}

	return trReason, nil
}

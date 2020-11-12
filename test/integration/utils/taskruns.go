// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"

	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/apis"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
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
	if len(trList.Items) == 0 {
		// Return a "NotFound" error so that we can use kerrors.IsNotFound on the returned error
		return nil, kerrors.NewNotFound(v1beta1.Resource("taskruns"), buildRunName)
	}

	if len(trList.Items) > 1 {
		return nil, fmt.Errorf("found %d TaskRuns for Buildrun %s", len(trList.Items), buildRunName)
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

	return "", fmt.Errorf("no Succeeded condition found for BuildRun %s", buildRunName)
}

// GetTRTillDesiredReason polls until a TaskRun matches a desired Reason
// or it exits if an error happen or a timeout is reach.
func (t *TestBuild) GetTRTillDesiredReason(buildRunName string, reason string) error {

	var (
		pollTRTillCompletion = func() (bool, error) {

			trReason, err := t.GetTRReason(buildRunName)
			// Not found means that we may not have a TaskRun for this BuildRun yet
			if kerrors.IsNotFound(err) {
				fmt.Printf("could not find TaskRun for BuildRun %s", buildRunName)
				return false, nil
			}
			if err != nil {
				return false, err
			}

			if trReason == reason {
				return true, nil
			}
			fmt.Printf("expecting TaskRun reason %s for BuildRun %s, got %s", reason, buildRunName, reason)
			return false, nil
		}
	)

	err := wait.PollImmediate(t.Interval, t.TimeOut, pollTRTillCompletion)
	if err != nil {
		return err
	}

	return nil
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"errors"
	"fmt"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"knative.dev/pkg/apis"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

// This class is intended to host all CRUD calls for testing TaskRuns CRDs resources

// GetTaskRunFromBuildRun retrieves an owned TaskRun based on the BuildRunName
func (t *TestBuild) GetTaskRunFromBuildRun(buildRunName string) (*pipelineapi.TaskRun, error) {
	taskRunLabelSelector := fmt.Sprintf("buildrun.shipwright.io/name=%s", buildRunName)

	trInterface := t.PipelineClientSet.TektonV1().TaskRuns(t.Namespace)

	trList, err := trInterface.List(t.Context, metav1.ListOptions{
		LabelSelector: taskRunLabelSelector,
	})
	if err != nil {
		return nil, err
	}

	if len(trList.Items) != 1 {
		return nil, fmt.Errorf("failed to find an owned TaskRun based on a Buildrun %s/%s name", t.Namespace, buildRunName)
	}

	return &trList.Items[0], nil
}

// UpdateTaskRun applies changes to a TaskRun object
func (t *TestBuild) UpdateTaskRun(name string, apply func(tr *pipelineapi.TaskRun)) (*pipelineapi.TaskRun, error) {
	var tr *pipelineapi.TaskRun
	var err error
	for i := 0; i < 5; i++ {
		tr, err = t.LookupTaskRun(types.NamespacedName{
			Namespace: t.Namespace,
			Name:      name,
		})
		if err != nil {
			return nil, err
		}

		apply(tr)

		tr, err = t.PipelineClientSet.TektonV1().TaskRuns(t.Namespace).Update(t.Context, tr, metav1.UpdateOptions{})
		if err == nil {
			return tr, nil
		}
		// retry the famous ""Operation cannot be fulfilled on taskruns.tekton.dev \"buildrun-test-build-225-xkw6k\": the object has been modified; please apply your changes to the latest version and try again" error
		if !apierrors.IsConflict(err) {
			return nil, err
		}
	}

	return nil, err
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
func (t *TestBuild) GetTRTillDesiredReason(buildRunName string, reason string) (trReason string, err error) {
	err = wait.PollUntilContextTimeout(t.Context, t.Interval, t.TimeOut, true, func(_ context.Context) (bool, error) {
		trReason, err = t.GetTRReason(buildRunName)
		if err != nil {
			return false, err
		}

		if trReason == reason {
			return true, nil
		}

		return false, nil
	})

	return
}

// DeleteTR deletes a TaskRun from a desired namespace
func (t *TestBuild) DeleteTR(name string) error {
	trInterface := t.PipelineClientSet.TektonV1().TaskRuns(t.Namespace)

	if err := trInterface.Delete(t.Context, name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

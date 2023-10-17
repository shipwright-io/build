// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

// This class is intended to host all CRUD calls for testing BuildRun CRDs resources

// CreateBR generates a BuildRun on the current test namespace
func (t *TestBuild) CreateBR(buildRun *v1alpha1.BuildRun) error {
	brInterface := t.BuildClientSet.ShipwrightV1alpha1().BuildRuns(t.Namespace)

	_, err := brInterface.Create(context.TODO(), buildRun, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// UpdateBR updates a BuildRun on the current test namespace
func (t *TestBuild) UpdateBR(buildRun *v1alpha1.BuildRun) error {
	brInterface := t.BuildClientSet.ShipwrightV1alpha1().BuildRuns(t.Namespace)
	_, err := brInterface.Update(context.TODO(), buildRun, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// GetBR retrieves a BuildRun from a desired namespace
// Deprecated: Use LookupBuildRun instead.
func (t *TestBuild) GetBR(name string) (*v1alpha1.BuildRun, error) {
	brInterface := t.BuildClientSet.ShipwrightV1alpha1().BuildRuns(t.Namespace)

	br, err := brInterface.Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return br, nil
}

// DeleteBR deletes a BuildRun from a desired namespace
func (t *TestBuild) DeleteBR(name string) error {
	brInterface := t.BuildClientSet.ShipwrightV1alpha1().BuildRuns(t.Namespace)

	if err := brInterface.Delete(context.TODO(), name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

// GetBRReason ...
func (t *TestBuild) GetBRReason(name string) (string, error) {
	br, err := t.GetBR(name)
	if err != nil {
		return "", err
	}
	cond := br.Status.GetCondition(v1alpha1.Succeeded)
	if cond == nil {
		return "", errors.New("BuildRun had no Succeeded condition")
	}
	return cond.Reason, nil
}

// GetBRTillCompletion returns a BuildRun that have a CompletionTime set.
// If the timeout is reached or it fails when retrieving the BuildRun it will
// stop polling and return
func (t *TestBuild) GetBRTillCompletion(name string) (*v1alpha1.BuildRun, error) {

	var (
		pollBRTillCompletion = func() (bool, error) {

			bInterface := t.BuildClientSet.ShipwrightV1alpha1().BuildRuns(t.Namespace)

			buildRun, err := bInterface.Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}
			if buildRun.Status.CompletionTime != nil {
				return true, nil
			}

			return false, nil
		}
	)

	brInterface := t.BuildClientSet.ShipwrightV1alpha1().BuildRuns(t.Namespace)

	err := wait.PollImmediate(t.Interval, t.TimeOut, pollBRTillCompletion)
	if err != nil {
		return nil, err
	}

	return brInterface.Get(context.TODO(), name, metav1.GetOptions{})
}

// GetBRTillNotFound waits for the buildrun to get deleted. It returns an error if BuildRun is not found
func (t *TestBuild) GetBRTillNotFound(name string, interval time.Duration, timeout time.Duration) (*v1alpha1.BuildRun, error) {

	var (
		GetBRTillNotFound = func() (bool, error) {

			bInterface := t.BuildClientSet.ShipwrightV1alpha1().BuildRuns(t.Namespace)
			_, err := bInterface.Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil && apierrors.IsNotFound(err) {
				return true, err
			}
			return false, nil
		}
	)

	brInterface := t.BuildClientSet.ShipwrightV1alpha1().BuildRuns(t.Namespace)

	err := wait.PollImmediate(interval, timeout, GetBRTillNotFound)
	if err != nil {
		return nil, err
	}

	return brInterface.Get(context.TODO(), name, metav1.GetOptions{})
}

// GetBRTillNotOwner returns a BuildRun that has not an owner.
// If the timeout is reached or it fails when retrieving the BuildRun it will
// stop polling and return
func (t *TestBuild) GetBRTillNotOwner(name string, owner string) (*v1alpha1.BuildRun, error) {

	brInterface := t.BuildClientSet.ShipwrightV1alpha1().BuildRuns(t.Namespace)

	var (
		pollBRTillNotOwner = func() (bool, error) {

			buildRun, err := brInterface.Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}

			for _, ownerReference := range buildRun.OwnerReferences {
				if ownerReference.Name == owner {
					return false, nil
				}
			}

			return true, nil
		}
	)

	if err := wait.PollImmediate(t.Interval, t.TimeOut, pollBRTillNotOwner); err != nil {
		return nil, err
	}

	return brInterface.Get(context.TODO(), name, metav1.GetOptions{})
}

// GetBRTillOwner returns a BuildRun that has an owner.
// If the timeout is reached or it fails when retrieving the BuildRun it will
// stop polling and return
func (t *TestBuild) GetBRTillOwner(name string, owner string) (*v1alpha1.BuildRun, error) {

	brInterface := t.BuildClientSet.ShipwrightV1alpha1().BuildRuns(t.Namespace)

	var (
		pollBRTillOwner = func() (bool, error) {

			buildRun, err := brInterface.Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}

			for _, ownerReference := range buildRun.OwnerReferences {
				if ownerReference.Name == owner {
					return true, nil
				}
			}

			return false, nil
		}
	)

	if err := wait.PollImmediate(t.Interval, t.TimeOut, pollBRTillOwner); err != nil {
		return nil, err
	}

	return brInterface.Get(context.TODO(), name, metav1.GetOptions{})
}

// GetBRTillStartTime returns a BuildRun that have a StartTime set.
// If the timeout is reached or it fails when retrieving the BuildRun it will
// stop polling and return
func (t *TestBuild) GetBRTillStartTime(name string) (*v1alpha1.BuildRun, error) {

	var (
		pollBRTillCompletion = func() (bool, error) {

			bInterface := t.BuildClientSet.ShipwrightV1alpha1().BuildRuns(t.Namespace)

			buildRun, err := bInterface.Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}
			if buildRun.Status.StartTime != nil {
				return true, nil
			}

			// early exit
			if buildRun.Status.CompletionTime != nil {
				if buildRunJSON, err := json.Marshal(buildRun); err == nil {
					return false, fmt.Errorf("buildrun is completed: %s", buildRunJSON)
				}

				return false, fmt.Errorf("buildrun is completed")
			}

			return false, nil
		}
	)

	brInterface := t.BuildClientSet.ShipwrightV1alpha1().BuildRuns(t.Namespace)

	err := wait.PollImmediate(t.Interval, t.TimeOut, pollBRTillCompletion)
	if err != nil {
		return nil, err
	}

	return brInterface.Get(context.TODO(), name, metav1.GetOptions{})
}

// GetBRTillDesiredReason polls until a BuildRun gets a particular Reason
// it exit if an error happens or the timeout is reached
func (t *TestBuild) GetBRTillDesiredReason(buildRunname string, reason string) (currentReason string, err error) {
	err = wait.PollImmediate(t.Interval, t.TimeOut, func() (bool, error) {
		currentReason, err = t.GetBRReason(buildRunname)
		if err != nil {
			return false, err
		}
		if currentReason == reason {
			return true, nil
		}

		return false, nil
	})

	return
}

// GetBRTillDeletion polls until a BuildRun is not found, it returns
// if a timeout is reached
func (t *TestBuild) GetBRTillDeletion(name string) (bool, error) {

	var (
		pollBRTillCompletion = func() (bool, error) {

			bInterface := t.BuildClientSet.ShipwrightV1alpha1().BuildRuns(t.Namespace)

			_, err := bInterface.Get(context.TODO(), name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		}
	)

	err := wait.PollImmediate(t.Interval, t.TimeOut, pollBRTillCompletion)
	if err != nil {
		return false, err
	}

	return true, nil
}

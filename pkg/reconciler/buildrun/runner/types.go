// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"context"

	"github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BuildRunner is an abstraction of any Kubernetes object that is capable running a build.
type BuildRunner interface {

	// GetCreationTimestamp returns the time that the runner object was created.
	GetCreationTimestamp() metav1.Time

	// GetCompletionTime returns the time that the runner completed its execution.
	GetCompletionTime() *metav1.Time

	// GetObject returns the kubernetes Object that runs the build.
	GetObject() client.Object

	// GetPodCreationTime returns the creation time of the pod that runs the build.
	GetPodCreationTime(ctx context.Context, client client.Client) *metav1.Time

	// GetPodInitFinishedTime returns the time at which all initialization tasks of the build completed.
	GetPodInitFinishedTime(ctx context.Context, client client.Client) *metav1.Time

	// GetStartTime returns the time that the runner started its execution.
	GetStartTime() *metav1.Time

	// Validate ensures the runner object is ready to be executed.
	Validate(ctx context.Context, client client.Client) *BuildRunnerValidationError

	// IsCancelled indicates if the build runner execution was cancelled.
	IsCancelled() bool

	// IsCompleted indicates if the build runner execution completed.
	IsCompleted() bool

	// Cancel issues a request to stop the build runner's execution.
	Cancel(ctx context.Context, client client.Client) error

	// SyncBuildRunStatus updates the BuildRun status with the BuildRunner's state.
	SyncBuildRunStatus(ctx context.Context, client client.Client, buildRun *v1beta1.BuildRun) error
}

// BuildRunnerFactory represents an entity that creates BuildRunner instances, and converts Kubernetes
// objects to their BuildRunner representations.
type BuildRunnerFactory interface {

	// CreateBuildRunner creates an instance of the factory's BuildRunner from the provided build
	// service account, strategy, Build, and BuildRun objects.
	CreateBuildRunner(sa *corev1.ServiceAccount, strategy v1beta1.BuilderStrategy,
		build *v1beta1.Build, buildRun *v1beta1.BuildRun) (BuildRunner, error)

	// ConvertToBuildRunner converts the provided Object to a BuildRunner instance. An error should
	// be raised if the factory instance does not support the underlying object type.
	ConvertToBuildRunner(obj client.Object) (BuildRunner, error)
}

// BuildRunnerValidationError is the error type for BuildRunner validation errors.
type BuildRunnerValidationError struct {
	// Terminal indicates if the build should fail because of this validation error.
	Terminal bool
	// Message is the message that should be logged or reported in the build's status.
	Message string
	// ReasonCode is a CamelCase shorthand that explains why the validation error occurred.
	ReasonCode string
}

// Error returns the message of the BuildRunnerValidationError.
func (e *BuildRunnerValidationError) Error() string {
	return e.Message
}

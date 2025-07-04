// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun

import (
	"context"
	"fmt"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"knative.dev/pkg/apis"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
)

// ImageBuildRunner defines an interface for building a container image.
type ImageBuildRunner interface {
	// GetName returns the name of the build runner.
	GetName() string
	// GetLabels returns the labels of the build runner.
	GetLabels() map[string]string
	// GetCreationTimestamp returns the creation timestamp of the build runner.
	GetCreationTimestamp() metav1.Time
	// GetResults returns the results of the build runner.
	GetResults() []pipelineapi.TaskRunResult
	// GetCondition returns the condition of the build runner.
	GetCondition(conditionType apis.ConditionType) *apis.Condition
	// GetStartTime returns the start time of the build runner.
	GetStartTime() *metav1.Time
	// GetCompletionTime returns the completion time of the build runner.
	GetCompletionTime() *metav1.Time
	// GetPodName returns the pod name of the build runner.
	GetPodName() string
	// IsCancelled returns true if the build runner is cancelled.
	IsCancelled() bool
	// GetObject returns the underlying client.Object for owner reference operations.
	GetObject() client.Object
}

// ImageBuildRunnerFactory defines methods for creating and manipulating ImageBuildRunners.
type ImageBuildRunnerFactory interface {
	// NewImageBuildRunner creates a new empty ImageBuildRunner.
	NewImageBuildRunner() ImageBuildRunner

	// CreateImageBuildRunner creates an ImageBuildRunner instance from build configuration. It does not create the ImageBuildRunner in the API server.
	CreateImageBuildRunner(ctx context.Context, cfg *config.Config, serviceAccount *corev1.ServiceAccount, strategy buildv1beta1.BuilderStrategy, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun, scheme *runtime.Scheme, setOwnerRef setOwnerReferenceFunc) (ImageBuildRunner, error)

	// GetImageBuildRunner retrieves an ImageBuildRunner from the API server.
	GetImageBuildRunner(ctx context.Context, client client.Client, namespacedName types.NamespacedName) (ImageBuildRunner, error)

	// CreateImageBuildRunnerInCluster creates the ImageBuildRunner in the API server.
	CreateImageBuildRunnerInCluster(ctx context.Context, client client.Client, taskRunner ImageBuildRunner) error
}

// TektonTaskRunWrapper wraps pipelineapi.TaskRun to implement the ImageBuildRunner interface.
type TektonTaskRunWrapper struct {
	TaskRun *pipelineapi.TaskRun
}

// GetName returns the name of the TaskRun
func (t *TektonTaskRunWrapper) GetName() string {
	if t.TaskRun == nil {
		return ""
	}
	return t.TaskRun.Name
}

// GetLabels returns the labels of the TaskRun
func (t *TektonTaskRunWrapper) GetLabels() map[string]string {
	if t.TaskRun == nil {
		return nil
	}
	return t.TaskRun.Labels
}

// GetCreationTimestamp returns the creation timestamp of the TaskRun
func (t *TektonTaskRunWrapper) GetCreationTimestamp() metav1.Time {
	if t.TaskRun == nil {
		return metav1.Time{}
	}
	return t.TaskRun.CreationTimestamp
}

// GetResults returns the TaskRun results
func (t *TektonTaskRunWrapper) GetResults() []pipelineapi.TaskRunResult {
	if t.TaskRun == nil {
		return nil
	}
	return t.TaskRun.Status.Results
}

// GetCondition returns the condition with the specified type
func (t *TektonTaskRunWrapper) GetCondition(conditionType apis.ConditionType) *apis.Condition {
	if t.TaskRun == nil {
		return nil
	}
	return t.TaskRun.Status.GetCondition(conditionType)
}

// GetStartTime returns the start time of the TaskRun
func (t *TektonTaskRunWrapper) GetStartTime() *metav1.Time {
	if t.TaskRun == nil {
		return nil
	}
	return t.TaskRun.Status.StartTime
}

// GetCompletionTime returns the completion time of the TaskRun
func (t *TektonTaskRunWrapper) GetCompletionTime() *metav1.Time {
	if t.TaskRun == nil {
		return nil
	}
	return t.TaskRun.Status.CompletionTime
}

// GetPodName returns the pod name of the TaskRun
func (t *TektonTaskRunWrapper) GetPodName() string {
	if t.TaskRun == nil {
		return ""
	}
	return t.TaskRun.Status.PodName
}

// IsCancelled returns true if the TaskRun is cancelled
func (t *TektonTaskRunWrapper) IsCancelled() bool {
	if t.TaskRun == nil {
		return false
	}
	return t.TaskRun.IsCancelled()
}

// GetObject returns the underlying client.Object for owner reference operations
func (t *TektonTaskRunWrapper) GetObject() client.Object {
	return t.TaskRun
}

// TektonTaskRunImageBuildRunnerFactory implements ImageBuildRunnerFactory for Tekton TaskRuns
type TektonTaskRunImageBuildRunnerFactory struct{}

// NewImageBuildRunner creates a new empty TaskRunner
func (f *TektonTaskRunImageBuildRunnerFactory) NewImageBuildRunner() ImageBuildRunner {
	return &TektonTaskRunWrapper{
		TaskRun: &pipelineapi.TaskRun{},
	}
}

// CreateImageBuildRunner creates an ImageBuildRunner instance from build configuration. It does not create the ImageBuildRunner in the API server.
func (f *TektonTaskRunImageBuildRunnerFactory) CreateImageBuildRunner(ctx context.Context, cfg *config.Config, serviceAccount *corev1.ServiceAccount, strategy buildv1beta1.BuilderStrategy, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun, scheme *runtime.Scheme, setOwnerRef setOwnerReferenceFunc) (ImageBuildRunner, error) {
	generatedTaskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccount.Name, strategy)
	if err != nil {
		return nil, err
	}

	// Set OwnerReference for BuildRun and TaskRun
	if err := setOwnerRef(buildRun, generatedTaskRun, scheme); err != nil {
		return nil, err
	}

	return &TektonTaskRunWrapper{TaskRun: generatedTaskRun}, nil
}

// GetImageBuildRunner retrieves an ImageBuildRunner from the API server.
func (f *TektonTaskRunImageBuildRunnerFactory) GetImageBuildRunner(ctx context.Context, client client.Client, namespacedName types.NamespacedName) (ImageBuildRunner, error) {
	taskRun := &pipelineapi.TaskRun{}
	err := client.Get(ctx, namespacedName, taskRun)
	if err != nil {
		return nil, err
	}
	return &TektonTaskRunWrapper{TaskRun: taskRun}, nil
}

// CreateImageBuildRunnerInCluster creates an ImageBuildRunner in the API server.
func (f *TektonTaskRunImageBuildRunnerFactory) CreateImageBuildRunnerInCluster(ctx context.Context, client client.Client, taskRunner ImageBuildRunner) error {
	wrapper, ok := taskRunner.(*TektonTaskRunWrapper)
	if !ok {
		return fmt.Errorf("unsupported TaskRunner type")
	}
	return client.Create(ctx, wrapper.TaskRun)
}

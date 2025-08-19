// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun

import (
	"context"
	"encoding/json"
	"fmt"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"knative.dev/pkg/apis"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TektonPipelineRunWrapper wraps pipelineapi.PipelineRun to implement the ImageBuildRunner interface.
type TektonPipelineRunWrapper struct {
	PipelineRun *pipelineapi.PipelineRun
}

// GetName returns the name of the PipelineRun.
func (t *TektonPipelineRunWrapper) GetName() string {
	if t.PipelineRun == nil {
		return ""
	}
	return t.PipelineRun.Name
}

// GetLabels returns the labels of the PipelineRun.
func (t *TektonPipelineRunWrapper) GetLabels() map[string]string {
	if t.PipelineRun == nil {
		return nil
	}
	return t.PipelineRun.Labels
}

// GetCreationTimestamp returns the creation timestamp of the PipelineRun.
func (t *TektonPipelineRunWrapper) GetCreationTimestamp() metav1.Time {
	if t.PipelineRun == nil {
		return metav1.Time{}
	}
	return t.PipelineRun.CreationTimestamp
}

// GetResults returns the PipelineRun results converted to TaskRun results.
func (t *TektonPipelineRunWrapper) GetResults() []pipelineapi.TaskRunResult {
	if t.PipelineRun == nil {
		return nil
	}
	var taskRunResults []pipelineapi.TaskRunResult
	for _, prResult := range t.PipelineRun.Status.Results {
		taskRunResults = append(taskRunResults, pipelineapi.TaskRunResult{
			Name:  prResult.Name,
			Value: prResult.Value,
		})
	}
	return taskRunResults
}

// GetCondition returns the condition with the specified type.
func (t *TektonPipelineRunWrapper) GetCondition(conditionType apis.ConditionType) *apis.Condition {
	if t.PipelineRun == nil {
		return nil
	}
	return t.PipelineRun.Status.GetCondition(conditionType)
}

// GetStartTime returns the start time of the PipelineRun.
func (t *TektonPipelineRunWrapper) GetStartTime() *metav1.Time {
	if t.PipelineRun == nil {
		return nil
	}
	return t.PipelineRun.Status.StartTime
}

// GetCompletionTime returns the completion time of the PipelineRun.
func (t *TektonPipelineRunWrapper) GetCompletionTime() *metav1.Time {
	if t.PipelineRun == nil {
		return nil
	}
	return t.PipelineRun.Status.CompletionTime
}

// GetPodName returns the pod name of the PipelineRun's first TaskRun.
func (t *TektonPipelineRunWrapper) GetPodName() string {
	if t.PipelineRun == nil || len(t.PipelineRun.Status.ChildReferences) == 0 {
		return ""
	}
	// For PipelineRuns, we need to get the pod name from the first TaskRun in ChildReferences.
	// Since we only have one task in our PipelineRun, we can use the first ChildReference.
	// The pod name will be the same as the TaskRun name with a suffix.
	firstTaskRef := t.PipelineRun.Status.ChildReferences[0]
	return firstTaskRef.Name + "-pod"
}

// IsCancelled returns true if the PipelineRun is cancelled.
func (t *TektonPipelineRunWrapper) IsCancelled() bool {
	if t.PipelineRun == nil {
		return false
	}
	return t.PipelineRun.IsCancelled()
}

// Cancel cancels the PipelineRun by setting its status to cancelled.
func (t *TektonPipelineRunWrapper) Cancel(ctx context.Context, c client.Client) error {
	if t.PipelineRun == nil {
		return fmt.Errorf("underlying PipelineRun does not exist")
	}

	payload := []patchStringValue{{
		Op:    "replace",
		Path:  "/spec/status",
		Value: pipelineapi.PipelineRunSpecStatusCancelled,
	}}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	patch := client.RawPatch(types.JSONPatchType, data)

	trueParam := true
	patchOpt := client.PatchOptions{
		Raw: &metav1.PatchOptions{
			Force: &trueParam,
		},
	}
	return c.Patch(ctx, t.PipelineRun, patch, &patchOpt)
}

// GetObject returns the underlying client.Object for owner reference operations.
func (t *TektonPipelineRunWrapper) GetObject() client.Object {
	return t.PipelineRun
}

// CheckVolumesExist checks if the volumes referenced by the PipelineRun exist.
// For PipelineRuns, we need to check volumes in the underlying TaskRun.
func (t *TektonPipelineRunWrapper) CheckVolumesExist(ctx context.Context, client client.Client) error {
	if t.PipelineRun == nil {
		return fmt.Errorf("underlying PipelineRun does not exist")
	}

	// For PipelineRuns, we need to get the underlying TaskRun to check volumes
	// This is a limitation of the current design - PipelineRuns don't have direct volume access
	// We could either:
	// 1. Skip volume checking for PipelineRuns (current approach)
	// 2. Get the TaskRun from ChildReferences and check its volumes
	// 3. Extract volume information from the PipelineRun spec

	// For now, we'll skip volume checking for PipelineRuns since the volumes
	// are defined in the embedded TaskSpec and should be valid
	return nil
}

// GetExecutorKind returns the kind of executor.
func (t *TektonPipelineRunWrapper) GetExecutorKind() string {
	return "PipelineRun"
}

// GetUnderlyingTaskRun returns nil since this is not a TaskRun-based runner.
func (t *TektonPipelineRunWrapper) GetUnderlyingTaskRun() *pipelineapi.TaskRun {
	return nil
}

// GetUnderlyingPipelineRun returns the underlying PipelineRun.
func (t *TektonPipelineRunWrapper) GetUnderlyingPipelineRun() *pipelineapi.PipelineRun {
	return t.PipelineRun
}

// TektonPipelineRunImageBuildRunnerFactory implements ImageBuildRunnerFactory for Tekton PipelineRuns.
type TektonPipelineRunImageBuildRunnerFactory struct{}

// NewImageBuildRunner creates a new empty ImageBuildRunner for a PipelineRun.
func (f *TektonPipelineRunImageBuildRunnerFactory) NewImageBuildRunner() ImageBuildRunner {
	return &TektonPipelineRunWrapper{
		PipelineRun: &pipelineapi.PipelineRun{},
	}
}

// CreateImageBuildRunner creates an ImageBuildRunner instance from build configuration.
func (f *TektonPipelineRunImageBuildRunnerFactory) CreateImageBuildRunner(cfg *config.Config, serviceAccount *corev1.ServiceAccount, strategy buildv1beta1.BuilderStrategy, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun, scheme *runtime.Scheme, setOwnerRef setOwnerReferenceFunc) (ImageBuildRunner, error) {
	generatedPipelineRun, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccount.Name, strategy)
	if err != nil {
		return nil, err
	}

	if err := setOwnerRef(buildRun, generatedPipelineRun, scheme); err != nil {
		return nil, err
	}

	return &TektonPipelineRunWrapper{PipelineRun: generatedPipelineRun}, nil
}

// GetImageBuildRunner retrieves an ImageBuildRunner from the API server.
func (f *TektonPipelineRunImageBuildRunnerFactory) GetImageBuildRunner(ctx context.Context, client client.Client, namespacedName types.NamespacedName) (ImageBuildRunner, error) {
	pipelineRun := &pipelineapi.PipelineRun{}
	err := client.Get(ctx, namespacedName, pipelineRun)
	if err != nil {
		return nil, err
	}
	return &TektonPipelineRunWrapper{PipelineRun: pipelineRun}, nil
}

// CreateImageBuildRunnerInCluster creates the ImageBuildRunner in the API server.
func (f *TektonPipelineRunImageBuildRunnerFactory) CreateImageBuildRunnerInCluster(ctx context.Context, client client.Client, runner ImageBuildRunner) error {
	wrapper, ok := runner.(*TektonPipelineRunWrapper)
	if !ok {
		return fmt.Errorf("unsupported ImageBuildRunner type")
	}
	return client.Create(ctx, wrapper.PipelineRun)
}

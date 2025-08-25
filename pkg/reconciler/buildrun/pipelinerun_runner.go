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

// GetExecutorKind returns the kind of executor.
func (t *TektonPipelineRunWrapper) GetExecutorKind() string {
	return "PipelineRun"
}

// GetUnderlyingTaskRun returns the generated TaskRun from the PipelineRun for volume checking.
func (t *TektonPipelineRunWrapper) GetUnderlyingTaskRun() *pipelineapi.TaskRun {
	if t.PipelineRun == nil {
		return nil
	}

	// For PipelineRuns, we need to create a TaskRun from the embedded TaskSpec
	// This is a simplified approach - in a full implementation, we would extract the TaskRun properly
	if t.PipelineRun.Spec.PipelineSpec != nil && len(t.PipelineRun.Spec.PipelineSpec.Tasks) > 0 {
		task := t.PipelineRun.Spec.PipelineSpec.Tasks[0]
		if task.TaskSpec != nil {
			return &pipelineapi.TaskRun{
				Spec: pipelineapi.TaskRunSpec{
					TaskSpec: &task.TaskSpec.TaskSpec,
					Params:   task.Params,
				},
			}
		}
	}

	return nil
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
func (f *TektonPipelineRunImageBuildRunnerFactory) CreateImageBuildRunner(ctx context.Context, client client.Client, cfg *config.Config, serviceAccount *corev1.ServiceAccount, strategy buildv1beta1.BuilderStrategy, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun, scheme *runtime.Scheme, setOwnerRef setOwnerReferenceFunc) (ImageBuildRunner, error) {
	generatedPipelineRun, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccount.Name, strategy)
	if err != nil {
		if updateErr := resources.UpdateConditionWithFalseStatus(ctx, client, buildRun, err.Error(), resources.ConditionTaskRunGenerationFailed); updateErr != nil {
			return nil, resources.HandleError("failed to create pipelinerun runtime object", err, updateErr)
		}

		return nil, err
	}

	if err := setOwnerRef(buildRun, generatedPipelineRun, scheme); err != nil {
		if updateErr := resources.UpdateConditionWithFalseStatus(ctx, client, buildRun, err.Error(), resources.ConditionSetOwnerReferenceFailed); updateErr != nil {
			return nil, resources.HandleError("failed to create pipelinerun runtime object", err, updateErr)
		}

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

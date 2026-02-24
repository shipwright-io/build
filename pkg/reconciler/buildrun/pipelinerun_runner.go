// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun

import (
	"context"
	"fmt"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"knative.dev/pkg/apis"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
// For PipelineRuns, we need to extract results from the underlying TaskRuns
// since results are typically written by individual TaskRuns, not by the PipelineRun itself.
func (t *TektonPipelineRunWrapper) GetResults(ctx context.Context, client client.Client) []pipelineapi.TaskRunResult {
	if t.PipelineRun == nil {
		return nil
	}

	var taskRunResults []pipelineapi.TaskRunResult

	// First, check if the PipelineRun has its own results
	for _, prResult := range t.PipelineRun.Status.Results {
		taskRunResults = append(taskRunResults, pipelineapi.TaskRunResult{
			Name:  prResult.Name,
			Value: prResult.Value,
		})
	}

	// If no PipelineRun results exist, extract results from underlying TaskRuns
	if len(taskRunResults) == 0 && len(t.PipelineRun.Status.ChildReferences) > 0 {
		// Extract results from all TaskRuns in the PipelineRun
		for _, childRef := range t.PipelineRun.Status.ChildReferences {
			if childRef.Kind == "TaskRun" {
				taskRun := &pipelineapi.TaskRun{}
				taskRunName := types.NamespacedName{
					Namespace: t.PipelineRun.Namespace,
					Name:      childRef.Name,
				}

				if err := client.Get(ctx, taskRunName, taskRun); err != nil {
					// Log error but continue with other TaskRuns
					continue
				}

				// Convert TaskRun results to TaskRunResult format
				for _, result := range taskRun.Status.Results {
					taskRunResults = append(taskRunResults, pipelineapi.TaskRunResult{
						Name:  result.Name,
						Value: result.Value,
					})
				}
			}
		}
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

func (t *TektonPipelineRunWrapper) GetPodName() string {
	// For PipelineRuns, we cannot reliably determine the pod name without fetching the TaskRun
	// due to potential name truncation in Tekton. The reconcile function handles this case by
	// fetching the TaskRun and using its Status.PodName for metrics collection (pod ramp-up duration).
	// Callers should use GetUnderlyingTaskRuns() to get the actual TaskRun and access its Status.PodName field.
	return ""
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

	// patching using server-side apply
	u := &unstructured.Unstructured{}
	u.SetAPIVersion("tekton.dev/v1")
	u.SetKind("PipelineRun")
	u.SetNamespace(t.PipelineRun.Namespace)
	u.SetName(t.PipelineRun.Name)
	if err := unstructured.SetNestedField(u.Object, pipelineapi.PipelineRunSpecStatusCancelled, "spec", "status"); err != nil {
		return err
	}

	return c.Patch(
		ctx,
		u,
		client.Apply,
		client.FieldOwner("shipwright-build-controller"),
		client.ForceOwnership,
	)
}

// GetObject returns the underlying client.Object for owner reference operations.
func (t *TektonPipelineRunWrapper) GetObject() client.Object {
	return t.PipelineRun
}

// GetExecutorKind returns the kind of executor.
func (t *TektonPipelineRunWrapper) GetExecutorKind() string {
	return "PipelineRun"
}

// GetUnderlyingTaskRuns returns the actual TaskRun from the child references in the PipelineRun.
func (t *TektonPipelineRunWrapper) GetUnderlyingTaskRuns(client client.Client) ([]*pipelineapi.TaskRun, error) {
	if t.PipelineRun == nil {
		return nil, fmt.Errorf("underlying PipelineRun does not exist")
	}

	// If no ChildReferences exist yet, return an empty slice to allow reconciliation to continue.
	if len(t.PipelineRun.Status.ChildReferences) == 0 {
		return []*pipelineapi.TaskRun{}, nil
	}

	var taskRuns []*pipelineapi.TaskRun
	for _, childRef := range t.PipelineRun.Status.ChildReferences {
		// Ensure the child is a TaskRun before attempting to fetch it.
		if childRef.Kind != "TaskRun" {
			continue
		}

		taskRun := &pipelineapi.TaskRun{}
		err := client.Get(context.Background(), types.NamespacedName{
			Name:      childRef.Name,
			Namespace: t.PipelineRun.Namespace,
		}, taskRun)

		if err != nil {
			// A missing TaskRun is a critical error.
			return nil, fmt.Errorf("failed to fetch TaskRun %s: %w", childRef.Name, err)
		}
		taskRuns = append(taskRuns, taskRun)
	}

	return taskRuns, nil
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
		if updateErr := resources.UpdateConditionWithFalseStatus(ctx, client, buildRun, err.Error(), resources.ConditionPipelineRunGenerationFailed); updateErr != nil {
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

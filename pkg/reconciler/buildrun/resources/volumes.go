// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func namespacedName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
}

// CheckVolumesExist checks if Secret, ConfigMap, or Projected volumes referenced in the slice exist in the namespace.
func CheckVolumesExist(ctx context.Context, client client.Client, namespace string, volumes []corev1.Volume) error {
	for _, volume := range volumes {
		var (
			err  error
			name string
		)

		switch {
		case volume.Secret != nil:
			secret := corev1.Secret{}
			name = volume.Secret.SecretName
			err = client.Get(ctx, namespacedName(name, namespace), &secret)
		case volume.ConfigMap != nil:
			configMap := corev1.ConfigMap{}
			name = volume.ConfigMap.Name
			err = client.Get(ctx, namespacedName(name, namespace), &configMap)
		case volume.Projected != nil:
			for _, projection := range volume.Projected.Sources {
				if projection.ConfigMap != nil {
					configMap := corev1.ConfigMap{}
					name = projection.ConfigMap.Name
					err = client.Get(ctx, namespacedName(name, namespace), &configMap)
				}
				if err == nil && projection.Secret != nil {
					secret := corev1.Secret{}
					name = projection.Secret.Name
					err = client.Get(ctx, namespacedName(name, namespace), &secret)
				}
				if err != nil {
					break
				}
			}
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// CheckTaskRunVolumesExist tries to find some of the volumes referenced by the BuildRun with all the
// overrides. If some secret or configmap does not exist in the namespace, function returns error
// describing the missing resource
func CheckTaskRunVolumesExist(ctx context.Context, client client.Client, taskRun *pipelineapi.TaskRun) error {
	if taskRun == nil || taskRun.Spec.TaskSpec == nil {
		return nil
	}
	return CheckVolumesExist(ctx, client, taskRun.Namespace, taskRun.Spec.TaskSpec.Volumes)
}

// CheckPipelineRunVolumesExist checks that all volumes referenced in the PipelineRun tasks exist.
func CheckPipelineRunVolumesExist(ctx context.Context, client client.Client, pipelineRun *pipelineapi.PipelineRun) error {
	if pipelineRun == nil {
		return nil
	}

	for _, task := range pipelineRun.Spec.PipelineSpec.Tasks {
		if task.TaskSpec != nil {
			if err := CheckVolumesExist(ctx, client, pipelineRun.Namespace, task.TaskSpec.TaskSpec.Volumes); err != nil {
				return err
			}
		}
	}

	for _, task := range pipelineRun.Spec.PipelineSpec.Finally {
		if task.TaskSpec != nil {
			if err := CheckVolumesExist(ctx, client, pipelineRun.Namespace, task.TaskSpec.TaskSpec.Volumes); err != nil {
				return err
			}
		}
	}

	return nil
}


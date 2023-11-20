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

// CheckTaskRunVolumesExist tries to find some of the volumes referenced by the BuildRun with all the
// overrides. If some secret or configmap does not exist in the namespace, function returns error
// describing the missing resource
func CheckTaskRunVolumesExist(ctx context.Context, client client.Client, taskRun *pipelineapi.TaskRun) error {
	for _, volume := range taskRun.Spec.TaskSpec.Volumes {
		var (
			err  error
			name string
		)

		switch {
		case volume.Secret != nil:
			secret := corev1.Secret{}
			name = volume.Secret.SecretName
			err = client.Get(ctx, namespacedName(name, taskRun.Namespace), &secret)
		case volume.ConfigMap != nil:
			configMap := corev1.ConfigMap{}
			name = volume.ConfigMap.Name
			err = client.Get(ctx, namespacedName(name, taskRun.Namespace), &configMap)
		case volume.Projected != nil:
			for _, projection := range volume.Projected.Sources {
				if projection.ConfigMap != nil {
					configMap := corev1.ConfigMap{}
					name = projection.ConfigMap.Name
					err = client.Get(ctx, namespacedName(name, taskRun.Namespace), &configMap)
				}
				if err == nil && projection.Secret != nil {
					secret := corev1.Secret{}
					name = projection.Secret.Name
					err = client.Get(ctx, namespacedName(name, taskRun.Namespace), &secret)
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

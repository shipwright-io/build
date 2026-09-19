// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"

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


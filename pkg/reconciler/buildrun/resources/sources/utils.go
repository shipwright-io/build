// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

const (
	prefixParamsResultsVolumes = "shp-"

	paramSourceRoot = "source-root"
)

var (
	nonRoot         = pointer.Int64Ptr(1000)
	secretMountMode = pointer.Int32Ptr(256) // is 0400
)

func AppendSecretVolume(
	taskSpec *tektonv1beta1.TaskSpec,
	secretName string,
) {
	volumeName := prefixParamsResultsVolumes + secretName

	// ensure we do not add the secret twice
	for _, volume := range taskSpec.Volumes {
		if volume.VolumeSource.Secret != nil && volume.Name == volumeName {
			return
		}
	}

	// append volume for secret
	taskSpec.Volumes = append(taskSpec.Volumes, corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  secretName,
				DefaultMode: secretMountMode,
			},
		},
	})
}

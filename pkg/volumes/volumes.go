// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package volumes

import (
	"fmt"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
)

// TaskSpecVolumes creates a list of Volumes for the generated TaskSpec object and
// checks for some erroneous situations around volumes and volume mounts
func TaskSpecVolumes(
	existingVolumeMounts map[string]bool,
	strategyVolumes []buildv1beta1.BuildStrategyVolume,
	buildVolumes []buildv1beta1.BuildVolume,
	buildrunVolumes []buildv1beta1.BuildVolume,
) ([]corev1.Volume, error) {
	res := []corev1.Volume{}

	// first we merge build volumes into the strategy ones, next we merge
	// build run volumes into result of the previous merge.
	// eventual list of volumes will be added to the generated TaskSpec object
	volumes, err := MergeBuildVolumes(strategyVolumes, buildVolumes)
	if err != nil {
		return nil, err
	}
	volumes, err = MergeBuildVolumes(volumes, buildrunVolumes)
	if err != nil {
		return nil, err
	}

	for i := range volumes {
		v := volumes[i]

		if readOnly, ok := existingVolumeMounts[v.Name]; ok {
			// In case volume mount is not read only and
			// volume type is either secret or config map,
			// build should not be run, because this situation may lead
			// to errors
			if !readOnly && isReadOnlyVolume(&v) {
				return nil,
					fmt.Errorf("Volume Mount %q must be read only", v.Name)
			}
		}

		taskRunVolume := corev1.Volume{
			Name:         v.Name,
			VolumeSource: v.VolumeSource,
		}

		res = append(res, taskRunVolume)
	}

	return res, nil
}

func isReadOnlyVolume(strategyVolume *buildv1beta1.BuildStrategyVolume) bool {
	return strategyVolume.VolumeSource.ConfigMap != nil ||
		strategyVolume.VolumeSource.Secret != nil ||
		strategyVolume.VolumeSource.DownwardAPI != nil ||
		strategyVolume.VolumeSource.Projected != nil
}

// MergeBuildVolumes merges Build Volumes from one list into the other. It only allows to merge those that have property
// Overridable set to true. In case it is empty or false, it is not allowed to be overridden, so Volume cannot be merged
// Merging in this context means copying the VolumeSource from one object to the other.
func MergeBuildVolumes(into []buildv1beta1.BuildStrategyVolume, new []buildv1beta1.BuildVolume) ([]buildv1beta1.BuildStrategyVolume, error) {
	if len(new) == 0 && len(into) == 0 {
		return []buildv1beta1.BuildStrategyVolume{}, nil
	}
	if len(new) == 0 {
		return into, nil
	}

	mergeMap := make(map[string]buildv1beta1.BuildStrategyVolume)
	var errors []error

	for _, vol := range into {
		if _, ok := mergeMap[vol.Name]; ok {
			return nil, fmt.Errorf("BuildStrategy Volume %q is listed more than once", vol.Name)
		}
		mergeMap[vol.Name] = *vol.DeepCopy()
	}

	for _, merger := range new {
		original, ok := mergeMap[merger.Name]
		if !ok {
			errors = append(errors, fmt.Errorf("Build Volume %q is not found in the BuildStrategy", merger.Name))
			continue
		}

		// in case overridable is nil OR false (default is considered false)
		// then return error, otherwise means it is not nil AND true
		if original.Overridable == nil || !*original.Overridable {
			errors = append(errors, fmt.Errorf("Cannot override BuildVolume %q", original.Name))
			continue
		}

		original.VolumeSource = merger.VolumeSource
		mergeMap[merger.Name] = original
	}

	result := make([]buildv1beta1.BuildStrategyVolume, 0, len(mergeMap))
	for _, v := range mergeMap {
		result = append(result, v)
	}

	return result, kerrors.NewAggregate(errors)
}

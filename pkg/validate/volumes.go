// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"fmt"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

// BuildVolumes is used to validate volumes in the Build object
func BuildVolumes(strategyVolumes []buildv1alpha1.BuildStrategyVolume, buildVolumes []buildv1alpha1.BuildVolume) (bool, buildv1alpha1.BuildReason, string) {
	return validateVolumes(strategyVolumes, buildVolumes)
}

// BuildRunVolumes is used to validate volumes in the BuildRun object
func BuildRunVolumes(strategyVolumes []buildv1alpha1.BuildStrategyVolume, buildVolumes []buildv1alpha1.BuildVolume) (bool, string, string) {
	valid, reason, msg := validateVolumes(strategyVolumes, buildVolumes)
	return valid, string(reason), msg
}

// validateBuildVolumes validates build overriding the build strategy volumes. in case it tries
// to override the non-overridable volume, or volume that does not exist in the strategy, it is
// good to fail early
func validateVolumes(strategyVolumes []build.BuildStrategyVolume, buildVolumes []build.BuildVolume) (bool, buildv1alpha1.BuildReason, string) {
	strategyVolumesMap := toVolumeMap(strategyVolumes)

	for _, buildVolume := range buildVolumes {
		strategyVolume, ok := strategyVolumesMap[buildVolume.Name]
		if !ok {
			return false, buildv1alpha1.UndefinedVolume, fmt.Sprintf("Volume %q is not defined in the Strategy", buildVolume.Name)
		}

		// nil for overridable is equal to false
		if strategyVolume.Overridable == nil || !*strategyVolume.Overridable {
			return false, buildv1alpha1.VolumeNotOverridable, fmt.Sprintf("Volume %q is not overridable in the Strategy", buildVolume.Name)
		}
	}

	return true, "", ""
}

// toVolumeMap coverts slice of build strategy volumes to map of build strategy volumes, in order to later search them quickly by name
func toVolumeMap(strategyVolumes []build.BuildStrategyVolume) map[string]build.BuildStrategyVolume {
	res := make(map[string]build.BuildStrategyVolume)
	for _, vol := range strategyVolumes {
		res[vol.Name] = vol
	}
	return res
}

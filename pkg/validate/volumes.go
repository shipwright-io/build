// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"fmt"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

// BuildVolumes is used to validate volumes in the Build object
func BuildVolumes(strategyVolumes []buildapi.BuildStrategyVolume, buildVolumes []buildapi.BuildVolume) (bool, buildapi.BuildReason, string) {
	return validateVolumes(strategyVolumes, buildVolumes)
}

// BuildRunVolumes is used to validate volumes in the BuildRun object
func BuildRunVolumes(strategyVolumes []buildapi.BuildStrategyVolume, buildVolumes []buildapi.BuildVolume) (bool, string, string) {
	valid, reason, msg := validateVolumes(strategyVolumes, buildVolumes)
	return valid, string(reason), msg
}

// validateBuildVolumes validates build overriding the build strategy volumes. in case it tries
// to override the non-overridable volume, or volume that does not exist in the strategy, it is
// good to fail early
func validateVolumes(strategyVolumes []buildapi.BuildStrategyVolume, buildVolumes []buildapi.BuildVolume) (bool, buildapi.BuildReason, string) {
	strategyVolumesMap := toVolumeMap(strategyVolumes)

	for _, buildVolume := range buildVolumes {
		strategyVolume, ok := strategyVolumesMap[buildVolume.Name]
		if !ok {
			return false, buildapi.UndefinedVolume, fmt.Sprintf("Volume %q is not defined in the Strategy", buildVolume.Name)
		}

		// nil for overridable is equal to false
		if strategyVolume.Overridable == nil || !*strategyVolume.Overridable {
			return false, buildapi.VolumeNotOverridable, fmt.Sprintf("Volume %q is not overridable in the Strategy", buildVolume.Name)
		}
	}

	return true, "", ""
}

// toVolumeMap coverts slice of build strategy volumes to map of build strategy volumes, in order to later search them quickly by name
func toVolumeMap(strategyVolumes []buildapi.BuildStrategyVolume) map[string]buildapi.BuildStrategyVolume {
	res := make(map[string]buildapi.BuildStrategyVolume)
	for _, vol := range strategyVolumes {
		res[vol.Name] = vol
	}
	return res
}

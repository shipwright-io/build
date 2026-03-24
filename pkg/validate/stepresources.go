// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"fmt"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

// validateStepResources is the shared validation logic for step resource overrides.
// It validates that all step resource overrides reference steps that exist in the build strategy.
func validateStepResources(strategySteps []buildapi.Step, stepResources []buildapi.StepResourceOverride) (bool, buildapi.BuildReason, string) {
	if len(stepResources) == 0 {
		return true, "", ""
	}

	// Build a set of valid step names from the strategy
	validStepNames := make(map[string]bool)
	for _, step := range strategySteps {
		validStepNames[step.Name] = true
	}

	// Validate each step resource override
	for _, stepResource := range stepResources {
		if !validStepNames[stepResource.Name] {
			return false, buildapi.UndefinedStepResource,
				fmt.Sprintf("stepResources references step %q which does not exist in the build strategy", stepResource.Name)
		}
	}

	return true, "", ""
}

// BuildStepResources validates that all step resource overrides in the Build
// reference steps that exist in the build strategy.
func BuildStepResources(strategySteps []buildapi.Step, buildStepResources []buildapi.StepResourceOverride) (bool, buildapi.BuildReason, string) {
	return validateStepResources(strategySteps, buildStepResources)
}

// BuildRunStepResources validates that all step resource overrides in the BuildRun
// reference steps that exist in the build strategy.
func BuildRunStepResources(strategySteps []buildapi.Step, buildRunStepResources []buildapi.StepResourceOverride) (bool, string, string) {
	valid, reason, msg := validateStepResources(strategySteps, buildRunStepResources)
	return valid, string(reason), msg
}

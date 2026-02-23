// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"fmt"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

// validateStepResources is the shared validation logic for step resource overrides.
// It validates that all step resource overrides reference steps that exist in the build strategy.
func validateStepResources(strategySteps []buildv1beta1.Step, stepResources []buildv1beta1.StepResourceOverride) (bool, buildv1beta1.BuildReason, string) {
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
			return false, buildv1beta1.UndefinedStepResource,
				fmt.Sprintf("stepResources references step %q which does not exist in the build strategy", stepResource.Name)
		}
	}

	return true, "", ""
}

// BuildStepResources validates that all step resource overrides in the Build
// reference steps that exist in the build strategy.
func BuildStepResources(strategySteps []buildv1beta1.Step, buildStepResources []buildv1beta1.StepResourceOverride) (bool, buildv1beta1.BuildReason, string) {
	return validateStepResources(strategySteps, buildStepResources)
}

// BuildRunStepResources validates that all step resource overrides in the BuildRun
// reference steps that exist in the build strategy.
func BuildRunStepResources(strategySteps []buildv1beta1.Step, buildRunStepResources []buildv1beta1.StepResourceOverride) (bool, string, string) {
	valid, reason, msg := validateStepResources(strategySteps, buildRunStepResources)
	return valid, string(reason), msg
}
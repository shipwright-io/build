// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package steps

import (
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// UpdateSecurityContext updates the security context of a step based on the build strategy steps. If all build strategy steps run as the same user and group,
// then the step is configured to also run as this user and group. This ensures that the supporting steps run as the same user as the build strategy and file
// permissions created by source steps match the user that runs the build strategy steps.
func UpdateSecurityContext(step *tektonapi.Step, buildStrategySteps []buildapi.BuildStep) {
	if len(buildStrategySteps) == 0 {
		return
	}

	var runAsUser *int64
	var runAsGroup *int64

	for i, buildStrategyStep := range buildStrategySteps {
		if buildStrategyStep.SecurityContext == nil {
			return
		}

		if buildStrategyStep.SecurityContext.RunAsUser == nil || buildStrategyStep.SecurityContext.RunAsGroup == nil {
			return
		}

		if i > 0 && (*buildStrategyStep.SecurityContext.RunAsUser != *runAsUser || *buildStrategyStep.SecurityContext.RunAsGroup != *runAsGroup) {
			return
		}

		runAsUser = buildStrategyStep.SecurityContext.RunAsUser
		runAsGroup = buildStrategyStep.SecurityContext.RunAsGroup
	}

	if step.SecurityContext == nil {
		step.SecurityContext = &corev1.SecurityContext{}
	}

	step.SecurityContext.RunAsUser = runAsUser
	step.SecurityContext.RunAsGroup = runAsGroup
}

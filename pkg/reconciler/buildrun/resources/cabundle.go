// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/cabundle"
)

// hasEnvVarByName checks if an environment variable with the given name already exists
func hasEnvVarByName(envs []corev1.EnvVar, name string) bool {
	for _, env := range envs {
		if env.Name == name {
			return true
		}
	}
	return false
}

func applyCABundle(taskSpec *pipelineapi.TaskSpec, build *buildapi.Build, buildRun *buildapi.BuildRun) error {
	var ca *buildapi.CABundle

	if taskSpec == nil {
		return nil
	}

	switch {
	case buildRun.Spec.CABundle != nil:
		ca = buildRun.Spec.CABundle
	case build.Spec.CABundle != nil:
		ca = build.Spec.CABundle
	default:
		return nil
	}

	envVar := cabundle.NewEnvVar()
	volume := cabundle.NewVolume(ca)

	taskSpec.Volumes = append(taskSpec.Volumes, *volume)
	for n, step := range taskSpec.Steps {
		for _, e := range envVar {
			if !hasEnvVarByName(step.Env, e.Name) {
				taskSpec.Steps[n].Env = append(step.Env, e)
			}
		}
		taskSpec.Steps[n].VolumeMounts = append(step.VolumeMounts, cabundle.NewVolumeMount(volume)...)
	}

	return nil
}

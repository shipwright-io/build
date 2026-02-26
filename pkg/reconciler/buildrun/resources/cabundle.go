// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package resources

import (
	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/cabundle"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func addCertificates(taskRun *pipelineapi.TaskRun, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun) error {
	var ca *buildv1beta1.CABundle

	switch {
	case buildRun.Spec.CABundle != nil:
		ca = buildRun.Spec.CABundle
	case build.Spec.CABundle != nil:
		ca = build.Spec.CABundle
	default:
		return nil
	}

	v := cabundle.NewVolume(ca)
	taskRun.Spec.TaskSpec.Volumes = append(taskRun.Spec.TaskSpec.Volumes, *v)

	for n := range taskRun.Spec.TaskSpec.Steps {
		taskRun.Spec.TaskSpec.Steps[n].Env = append(taskRun.Spec.TaskSpec.Steps[n].Env, cabundle.NewEnvVar()...)
		taskRun.Spec.TaskSpec.Steps[n].VolumeMounts = append(taskRun.Spec.TaskSpec.Steps[n].VolumeMounts, cabundle.NewVolumeMount(v)...)
	}

	return nil
}

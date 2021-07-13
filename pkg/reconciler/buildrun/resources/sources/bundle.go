// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"fmt"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// AppendBundleStep appends the bundle step to the TaskSpec
func AppendBundleStep(
	cfg *config.Config,
	taskSpec *tektonv1beta1.TaskSpec,
	source buildv1alpha1.Source,
	name string,
) {
	// initialize the step from the template
	bundleStep := tektonv1beta1.Step{
		Container: *cfg.BundleContainerTemplate.DeepCopy(),
	}

	// add the build-specific details
	bundleStep.Container.Name = fmt.Sprintf("bundle-%s", name)
	bundleStep.Container.Args = []string{
		"--image", source.Container.Image,
		"--target", fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramSourceRoot),
	}

	// add credentials mount, if provided
	if source.Credentials != nil {
		AppendSecretVolume(taskSpec, source.Credentials.Name)

		secretMountPath := fmt.Sprintf("/workspace/%s-pull-secret", prefixParamsResultsVolumes)

		// define the volume mount on the container
		bundleStep.VolumeMounts = append(bundleStep.VolumeMounts, corev1.VolumeMount{
			Name:      SanitizeVolumeNameForSecretName(source.Credentials.Name),
			MountPath: secretMountPath,
			ReadOnly:  true,
		})

		// append the argument
		bundleStep.Container.Args = append(bundleStep.Container.Args,
			"--secret-path", secretMountPath,
		)
	}

	taskSpec.Steps = append(taskSpec.Steps, bundleStep)
}

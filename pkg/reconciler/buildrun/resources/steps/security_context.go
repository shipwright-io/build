// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package steps

import (
	"fmt"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	"k8s.io/utils/strings/slices"
)

const (
	// AnnotationSecurityContextGroup is an annotation set on the TaskRun and used in a downward volume to project a dynamic group file into a container
	AnnotationSecurityContextGroup = buildapi.BuildRunDomain + "/security-context-group"

	// AnnotationSecurityContextPasswd is an annotation set on the TaskRun and used in a downward volume to project a dynamic passwd file into a container
	AnnotationSecurityContextPasswd = buildapi.BuildRunDomain + "/security-context-passwd"

	// VolumeNameSecurityContext is used as a volume name for a downward volume to project a dynamic passwd file into a container
	VolumeNameSecurityContext = "shp-security-context"
)

// UpdateSecurityContext updates the security context of a step based on the build strategy steps. If all build strategy steps run as the same user and group,
// then the step is configured to also run as this user and group. This ensures that the supporting steps run as the same user as the build strategy and file
// permissions created by source steps match the user that runs the build strategy steps.
func UpdateSecurityContext(taskSpec *pipelineapi.TaskSpec, taskRunAnnotations map[string]string, buildStrategySteps []buildapi.Step, buildStrategySecurityContext *buildapi.BuildStrategySecurityContext) {
	if buildStrategySecurityContext == nil {
		return
	}

	buildStrategyStepNames := make([]string, len(buildStrategySteps))
	for i, buildStrategyStep := range buildStrategySteps {
		buildStrategyStepNames[i] = buildStrategyStep.Name
	}

	volumeAdded := false

	for i := range taskSpec.Steps {
		if taskSpec.Steps[i].SecurityContext == nil {
			taskSpec.Steps[i].SecurityContext = &corev1.SecurityContext{}
		}

		if slices.Contains(buildStrategyStepNames, taskSpec.Steps[i].Name) {
			// for strategy steps, we only overwrite if nothing is defined
			if taskSpec.Steps[i].SecurityContext.RunAsUser == nil {
				taskSpec.Steps[i].SecurityContext.RunAsUser = &buildStrategySecurityContext.RunAsUser
			}
			if taskSpec.Steps[i].SecurityContext.RunAsGroup == nil {
				taskSpec.Steps[i].SecurityContext.RunAsGroup = &buildStrategySecurityContext.RunAsGroup
			}
		} else {
			// for shipwright-managed steps, we overwrite the default from the configuration and mount /etc/group and /etc/passwd
			taskSpec.Steps[i].SecurityContext.RunAsUser = &buildStrategySecurityContext.RunAsUser
			taskSpec.Steps[i].SecurityContext.RunAsGroup = &buildStrategySecurityContext.RunAsGroup

			if !volumeAdded {
				taskRunAnnotations[AnnotationSecurityContextGroup] = fmt.Sprintf("shp:x:%d", buildStrategySecurityContext.RunAsGroup)
				taskRunAnnotations[AnnotationSecurityContextPasswd] = fmt.Sprintf("shp:x:%d:%d:shp:/shared-home:/sbin/nologin", buildStrategySecurityContext.RunAsUser, buildStrategySecurityContext.RunAsGroup)

				taskSpec.Volumes = append(taskSpec.Volumes, corev1.Volume{
					Name: VolumeNameSecurityContext,
					VolumeSource: corev1.VolumeSource{
						DownwardAPI: &corev1.DownwardAPIVolumeSource{
							DefaultMode: pointer.Int32(0444),

							Items: []corev1.DownwardAPIVolumeFile{{
								Path: "group",
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: fmt.Sprintf("metadata.annotations['%s']", AnnotationSecurityContextGroup),
								},
							}, {
								Path: "passwd",
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: fmt.Sprintf("metadata.annotations['%s']", AnnotationSecurityContextPasswd),
								},
							}},
						},
					},
				})

				volumeAdded = true
			}

			taskSpec.Steps[i].VolumeMounts = append(
				taskSpec.Steps[i].VolumeMounts,
				corev1.VolumeMount{
					Name:      VolumeNameSecurityContext,
					MountPath: "/etc/group",
					SubPath:   "group",
				}, corev1.VolumeMount{
					Name:      VolumeNameSecurityContext,
					MountPath: "/etc/passwd",
					SubPath:   "passwd",
				},
			)
		}
	}
}

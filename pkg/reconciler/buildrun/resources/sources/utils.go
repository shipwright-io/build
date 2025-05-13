// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"fmt"
	"regexp"
	"strings"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const (
	PrefixParamsResultsVolumes = "shp"

	paramSourceRoot = "source-root"
)

var (
	dnsLabel1123Forbidden = regexp.MustCompile("[^a-zA-Z0-9-]+")

	// secrets are volumes and volumes are mounted as root, as we run as non-root, we must use 0444 to allow non-root to read it
	secretMountMode = ptr.To[int32](0444)
)

// AppendSecretVolume checks if a volume for a secret already exists, if not it appends it to the TaskSpec
func AppendSecretVolume(
	taskSpec *pipelineapi.TaskSpec,
	secretName string,
) {
	volumeName := SanitizeVolumeNameForSecretName(secretName)

	// ensure we do not add the secret twice
	for _, volume := range taskSpec.Volumes {
		if volume.VolumeSource.Secret != nil && volume.Name == volumeName {
			return
		}
	}

	// append volume for secret
	taskSpec.Volumes = append(taskSpec.Volumes, corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  secretName,
				DefaultMode: secretMountMode,
			},
		},
	})
}

// SanitizeVolumeNameForSecretName creates the name of a Volume for a Secret
func SanitizeVolumeNameForSecretName(secretName string) string {
	// remove forbidden characters
	sanitizedName := dnsLabel1123Forbidden.ReplaceAllString(fmt.Sprintf("%s-%s", PrefixParamsResultsVolumes, secretName), "-")

	// ensure maximum length
	if len(sanitizedName) > 63 {
		sanitizedName = sanitizedName[:63]
	}

	// trim trailing dashes because the last character must be alphanumeric
	sanitizedName = strings.TrimSuffix(sanitizedName, "-")

	return sanitizedName
}

// SanitizeVolumeNameForStepName creates a sanitized volume name for a step
// Example: "build-and-push" -> "build-and-push", "source-git" -> "source-git"
// This ensures each step gets its own isolated writable home directory volume
func SanitizeVolumeNameForStepName(stepName string) string {
	// remove forbidden characters
	sanitizedName := dnsLabel1123Forbidden.ReplaceAllString(stepName, "-")

	// ensure maximum length (leave room for prefix "shp-writable-home-")
	maxStepNameLength := 63 - len("shp-writable-home-")
	if len(sanitizedName) > maxStepNameLength {
		sanitizedName = sanitizedName[:maxStepNameLength]
	}

	// trim trailing dashes because the last character must be alphanumeric
	sanitizedName = strings.TrimSuffix(sanitizedName, "-")

	// ensure we have at least some content
	if sanitizedName == "" {
		sanitizedName = "default"
	}

	return sanitizedName
}

func TaskResultName(sourceName, resultName string) string {
	return fmt.Sprintf("%s-source-%s-%s",
		PrefixParamsResultsVolumes,
		sourceName,
		resultName,
	)
}

func FindResultValue(results []pipelineapi.TaskRunResult, sourceName, resultName string) string {
	var name = TaskResultName(sourceName, resultName)
	for _, result := range results {
		if result.Name == name {
			return result.Value.StringVal
		}
	}

	return ""
}

// ensureVolume adds a volume to the TaskSpec if a volume with the same name does not already exist.
func ensureVolume(taskSpec *pipelineapi.TaskSpec, volume corev1.Volume) {
	for _, v := range taskSpec.Volumes {
		if v.Name == volume.Name {
			return
		}
	}
	taskSpec.Volumes = append(taskSpec.Volumes, volume)
}

// ensureVolumeMount adds a VolumeMount to a Step if a mount with the same name does not already exist.
func ensureVolumeMount(step *pipelineapi.Step, mount corev1.VolumeMount) {
	for _, m := range step.VolumeMounts {
		if m.Name == mount.Name {
			return
		}
	}
	step.VolumeMounts = append(step.VolumeMounts, mount)
}

// AppendWriteableVolumes configures writable volumes for a container step within a Tekton Task.
func AppendWriteableVolumes(
	taskSpec *pipelineapi.TaskSpec,
	targetStep *pipelineapi.Step,
	writableHomeDir string,
) {
	// Volume for /tmp (shared across containers as it's for temporary files)
	tmpVolumeName := "shp-tmp-data"
	ensureVolume(taskSpec, corev1.Volume{
		Name: tmpVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
	ensureVolumeMount(targetStep, corev1.VolumeMount{
		Name:      tmpVolumeName,
		MountPath: "/tmp",
	})

	// Volume for writable home directory (unique per container)
	// Generate a unique volume name based on the step name to ensure isolation
	homeVolumeName := fmt.Sprintf("shp-writable-home-%s", SanitizeVolumeNameForStepName(targetStep.Name))
	ensureVolume(taskSpec, corev1.Volume{
		Name: homeVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
	ensureVolumeMount(targetStep, corev1.VolumeMount{
		Name:      homeVolumeName,
		MountPath: writableHomeDir,
	})

	// Set HOME env var (override if present)
	found := false
	for i, env := range targetStep.Env {
		if env.Name == "HOME" {
			targetStep.Env[i].Value = writableHomeDir
			found = true
			break
		}
	}
	if !found {
		targetStep.Env = append(targetStep.Env, corev1.EnvVar{
			Name:  "HOME",
			Value: writableHomeDir,
		})
	}
}

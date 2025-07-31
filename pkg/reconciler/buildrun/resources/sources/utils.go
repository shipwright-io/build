// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"crypto/sha256"
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

// AppendWriteableVolumes configures writable volumes for a specific step in a Tekton Task.
// It ensures that these volumes are not shared with other steps in the same pod.
func AppendWriteableVolumes(
	taskSpec *pipelineapi.TaskSpec,
	targetStep *pipelineapi.Step,
) {
	// Define a custom, isolated path for temporary files and mount it.
	tmpDir := "/shp-tmp"
	addStepEmptyDirVolume(
		taskSpec,
		targetStep,
		generateVolumeName("shp-tmp-", targetStep.Name),
		tmpDir,
	)
	// Point the TMPDIR environment variable to the custom path.
	setEnvVar(targetStep, "TMPDIR", tmpDir)
}

// generateVolumeName creates a unique, DNS-1123 compliant volume name for a step.
// The function ensures uniqueness by appending a SHA256 hash of the original step name.
func generateVolumeName(prefix, stepName string) string {
	// Create the full name first, then sanitize it
	name := fmt.Sprintf("%s%s", prefix, stepName)

	// Convert to lowercase and remove forbidden characters
	sanitizedName := strings.ToLower(dnsLabel1123Forbidden.ReplaceAllString(name, "-"))

	// Remove both leading and trailing hyphens
	sanitizedName = strings.Trim(sanitizedName, "-")

	// Generate a short hash of the original stepName for uniqueness
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(stepName)))[:8]

	// Ensure maximum length, leaving space for the hash
	maxLength := 63 - len(hash) - 1 // -1 for the hyphen separator
	if len(sanitizedName) > maxLength {
		sanitizedName = sanitizedName[:maxLength]
	}

	// Combine sanitized name with hash
	result := fmt.Sprintf("%s-%s", sanitizedName, hash)

	return result
}

// addStepEmptyDirVolume creates a unique EmptyDir volume for a specific step and mounts it at the given path.
func addStepEmptyDirVolume(taskSpec *pipelineapi.TaskSpec, step *pipelineapi.Step, volumeName, mountPath string) {
	ensureVolume(taskSpec, corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	ensureVolumeMount(step, corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
	})
}

// setEnvVar sets or overrides an environment variable in a Step.
func setEnvVar(step *pipelineapi.Step, name, value string) {
	for i, env := range step.Env {
		if env.Name == name {
			// Override existing variable
			step.Env[i].Value = value
			return
		}
	}

	// Append new variable if it does not exist
	step.Env = append(step.Env, corev1.EnvVar{
		Name:  name,
		Value: value,
	})
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

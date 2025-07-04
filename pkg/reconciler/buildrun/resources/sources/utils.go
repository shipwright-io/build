// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shipwright-io/build/pkg/config"
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

// ensureEnvVar adds or updates an environment variable in a step.
func ensureEnvVar(step *pipelineapi.Step, name, value string) {
	for i, envVar := range step.Env {
		if envVar.Name == name {
			step.Env[i].Value = value
			return
		}
	}
	step.Env = append(step.Env, corev1.EnvVar{Name: name, Value: value})
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
// It uses a single shared volume for the home directory and its subdirectories (.docker, .ssh)
// A separate volume is used for /tmp.
func AppendWriteableVolumes(
	taskSpec *pipelineapi.TaskSpec,
	targetStep *pipelineapi.Step,
	cfg *config.Config,
) {
	// Determine the base writable home directory path from the configuration.
	homeDir := cfg.ContainersWritableDir.WritableHomeDir
	// Ensure the HOME environment variable is set correctly
	ensureEnvVar(targetStep, "HOME", homeDir)

	// Volume for the base /shared-home directory
	baseHomeVolumeName := "base-home-volume"
	ensureVolume(taskSpec, corev1.Volume{
		Name: baseHomeVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
	ensureVolumeMount(targetStep, corev1.VolumeMount{
		Name:      baseHomeVolumeName,
		MountPath: homeDir,
	})

	// Volume for /shared-home/.docker
	dockerVolumeName := "docker-config-volume"
	ensureVolume(taskSpec, corev1.Volume{
		Name: dockerVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
	ensureVolumeMount(targetStep, corev1.VolumeMount{
		Name:      dockerVolumeName,
		MountPath: filepath.Join(homeDir, ".docker"),
	})

	// Volume for /shared-home/.ssh
	sshVolumeName := "ssh-keys-volume"
	ensureVolume(taskSpec, corev1.Volume{
		Name: sshVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
	ensureVolumeMount(targetStep, corev1.VolumeMount{
		Name:      sshVolumeName,
		MountPath: filepath.Join(homeDir, ".ssh"),
	})

	// Volume for /tmp
	tmpVolumeName := "tmp-data"
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
}

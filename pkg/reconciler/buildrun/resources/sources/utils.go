// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"fmt"
	"regexp"
	"strings"

	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

const (
	prefixParamsResultsVolumes = "shp"

	paramSourceRoot = "source-root"
)

var (
	dnsLabel1123Forbidden = regexp.MustCompile("[^a-zA-Z0-9-]+")

	// secrets are volumes and volumes are mounted as root, as we run as non-root, we must use 0444 to allow non-root to read it
	secretMountMode = pointer.Int32(0444)
)

// AppendSecretVolume checks if a volume for a secret already exists, if not it appends it to the TaskSpec
func AppendSecretVolume(
	taskSpec *tektonv1beta1.TaskSpec,
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
	sanitizedName := dnsLabel1123Forbidden.ReplaceAllString(fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, secretName), "-")

	// ensure maximum length
	if len(sanitizedName) > 63 {
		sanitizedName = sanitizedName[:63]
	}

	// trim trailing dashes because the last character must be alphanumeric
	sanitizedName = strings.TrimSuffix(sanitizedName, "-")

	return sanitizedName
}

func findResultValue(results []tektonv1beta1.TaskRunResult, name string) string {
	for _, result := range results {
		if result.Name == name {
			return result.Value.StringVal
		}
	}

	return ""
}

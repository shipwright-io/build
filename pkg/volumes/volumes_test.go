// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package volumes_test

import (
	"encoding/json"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/volumes"
)

type volumeType string

const (
	emptyDirVT  volumeType = "EmptyDir"
	configMapVT volumeType = "ConfigMap"
	secretVT    volumeType = "Secret"
)

func createConfigMapVolumeSource(name string) *corev1.VolumeSource {
	return &corev1.VolumeSource{
		ConfigMap: &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: name,
			},
		},
	}
}

func createSecretVolumeSource(name string) *corev1.VolumeSource {
	return &corev1.VolumeSource{
		Secret: &corev1.SecretVolumeSource{
			SecretName: name,
		},
	}
}

func createEmptyDirVolumeSource() *corev1.VolumeSource {
	return &corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{},
	}
}

func createVolumeSource(vt volumeType, vsName string) *corev1.VolumeSource {
	var vs *corev1.VolumeSource
	switch vt {
	case configMapVT:
		vs = createConfigMapVolumeSource(vsName)
	case secretVT:
		vs = createSecretVolumeSource(vsName)
	case emptyDirVT:
		vs = createEmptyDirVolumeSource()
	}
	return vs
}

func createBuildStrategyVolume(name string, description string, vt volumeType, vsName string, overridable bool) buildapi.BuildStrategyVolume {
	vs := createVolumeSource(vt, vsName)

	var descr *string
	if len(description) > 0 {
		descr = &description
	}

	bv := buildapi.BuildStrategyVolume{
		Name:         name,
		Description:  descr,
		VolumeSource: *vs,
		Overridable:  &overridable,
	}
	return bv
}

func createBuildStrategyVolumeEmptyOverridable(name string, description string, vt volumeType, vsName string) buildapi.BuildStrategyVolume {
	vs := createVolumeSource(vt, vsName)

	var descr *string
	if len(description) > 0 {
		descr = &description
	}

	bv := buildapi.BuildStrategyVolume{
		Name:         name,
		Description:  descr,
		VolumeSource: *vs,
	}
	return bv
}

func createBuildVolume(name string, vt volumeType, vsName string) buildapi.BuildVolume {
	vs := createVolumeSource(vt, vsName)

	bv := buildapi.BuildVolume{
		Name:         name,
		VolumeSource: *vs,
	}
	return bv
}

func TestMergeVolumes(t *testing.T) {

	testingData := []struct {
		name      string
		into      []buildapi.BuildStrategyVolume
		mergers   []buildapi.BuildVolume
		expected  []buildapi.BuildStrategyVolume
		expectErr bool
	}{
		{
			name:      "both empty",
			into:      []buildapi.BuildStrategyVolume{},
			mergers:   []buildapi.BuildVolume{},
			expected:  []buildapi.BuildStrategyVolume{},
			expectErr: false,
		},
		{
			name: "mergers empty",
			into: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "ConfigMap", "my-config", true),
			},
			mergers: []buildapi.BuildVolume{},
			expected: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "ConfigMap", "my-config", true),
			},
			expectErr: false,
		},
		{
			name: "into empty must fail",
			into: []buildapi.BuildStrategyVolume{},
			mergers: []buildapi.BuildVolume{
				createBuildVolume("bvname", "ConfigMap", "my-config"),
			},
			expected: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "ConfigMap", "my-config", true),
			},
			expectErr: true,
		},
		{
			name: "override one emptyDir with secret",
			into: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "EmptyDir", "", true),
			},
			mergers: []buildapi.BuildVolume{
				createBuildVolume("bvname", "ConfigMap", "my-config"),
			},
			expected: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "ConfigMap", "my-config", true),
			},
			expectErr: false,
		},
		{
			name: "connot override - not overridable, expecting error",
			into: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "EmptyDir", "", false),
			},
			mergers: []buildapi.BuildVolume{
				createBuildVolume("bvname", "ConfigMap", "my-config"),
			},
			expected: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "ConfigMap", "my-config", true),
			},
			expectErr: true,
		},
		{
			name: "connot override - volume does not exist, must produce err",
			into: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "EmptyDir", "", true),
			},
			mergers: []buildapi.BuildVolume{
				createBuildVolume("bvname2", "ConfigMap", "my-config"),
			},
			expected: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "EmptyDir", "", true),
			},
			expectErr: true,
		},
		{
			name: "override second",
			into: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "EmptyDir", "", false),
				createBuildStrategyVolume("bvname2", "bv description 2", "ConfigMap", "config-name", true),
				createBuildStrategyVolume("bvname3", "bv description 3", "Secret", "very-secret-name", true),
			},
			mergers: []buildapi.BuildVolume{
				createBuildVolume("bvname2", "Secret", "secret-name"),
			},
			expected: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "EmptyDir", "", false),
				createBuildStrategyVolume("bvname2", "bv description 2", "Secret", "secret-name", true),
				createBuildStrategyVolume("bvname3", "bv description 3", "Secret", "very-secret-name", true),
			},
			expectErr: false,
		},
		{
			name: "override first",
			into: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "EmptyDir", "", true),
				createBuildStrategyVolume("bvname2", "bv description 2", "ConfigMap", "config-name", false),
				createBuildStrategyVolume("bvname3", "bv description 3", "Secret", "very-secret-name", false),
			},
			mergers: []buildapi.BuildVolume{
				createBuildVolume("bvname", "Secret", "secret-name"),
			},
			expected: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "Secret", "secret-name", true),
				createBuildStrategyVolume("bvname2", "bv description 2", "ConfigMap", "config-name", false),
				createBuildStrategyVolume("bvname3", "bv description 3", "Secret", "very-secret-name", false),
			},
			expectErr: false,
		},
		{
			name: "override third",
			into: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "EmptyDir", "", true),
				createBuildStrategyVolume("bvname2", "bv description 2", "ConfigMap", "config-name", false),
				createBuildStrategyVolume("bvname3", "bv description 3", "Secret", "very-secret-name", true),
			},
			mergers: []buildapi.BuildVolume{
				createBuildVolume("bvname3", "EmptyDir", ""),
			},
			expected: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "EmptyDir", "", true),
				createBuildStrategyVolume("bvname2", "bv description 2", "ConfigMap", "config-name", false),
				createBuildStrategyVolume("bvname3", "bv description 3", "EmptyDir", "", true),
			},
			expectErr: false,
		},
		{
			name: "override second and third",
			into: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "EmptyDir", "", false),
				createBuildStrategyVolume("bvname2", "bv description 2", "ConfigMap", "config-name", true),
				createBuildStrategyVolume("bvname3", "bv description 3", "Secret", "very-secret-name", true),
			},
			mergers: []buildapi.BuildVolume{
				createBuildVolume("bvname2", "Secret", "very-very-secret"),
				createBuildVolume("bvname3", "EmptyDir", ""),
			},
			expected: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolume("bvname", "bv description", "EmptyDir", "", false),
				createBuildStrategyVolume("bvname2", "bv description 2", "Secret", "very-very-secret", true),
				createBuildStrategyVolume("bvname3", "bv description 3", "EmptyDir", "", true),
			},
			expectErr: false,
		},
		{
			name: "empty overridable cant be ovirriden",
			into: []buildapi.BuildStrategyVolume{
				createBuildStrategyVolumeEmptyOverridable("bvname", "desc", "EmptyDir", ""),
				createBuildStrategyVolume("bvname2", "bv description 2", "ConfigMap", "config-name", true),
			},
			mergers: []buildapi.BuildVolume{
				createBuildVolume("bvname", "Secret", "very-very-secret"),
				createBuildVolume("bvname2", "Secret", "very-secret-2"),
			},
			expected:  []buildapi.BuildStrategyVolume{},
			expectErr: true,
		},
	}

	for _, td := range testingData {
		t.Run(td.name, func(t *testing.T) {
			res, err := volumes.MergeBuildVolumes(td.into, td.mergers)

			if (err != nil) != td.expectErr {
				t.Errorf("%s: expected error %v, got %v", td.name, td.expectErr, err)
			}

			// if we have been expecting err and if it happened, next checks should not be
			// checked
			if td.expectErr {
				return
			}

			// volumes can be out of order, so we should convert to map, check length and then
			// check that every expected volume exists in the actual merge result
			volMap := toVolMap(res)

			if len(volMap) != len(td.expected) {
				t.Errorf("Length is not correct for merge result: %d, expected %d", len(volMap), len(td.expected))
			}

			for _, expectedVol := range td.expected {
				actualVol, ok := volMap[expectedVol.Name]
				if !ok {
					resJson, _ := json.Marshal(res)
					t.Errorf("Expected Volume %q not found in merge result %v", expectedVol.Name, string(resJson))
				}

				if !reflect.DeepEqual(expectedVol, actualVol) {
					expJson, _ := json.Marshal(expectedVol)
					actualJson, _ := json.Marshal(actualVol)
					t.Errorf("Expected volume is not equal to actual vol, actual: %v, expected: %v",
						string(actualJson), string(expJson))
				}
			}
		})
	}
}

func toVolMap(expected []buildapi.BuildStrategyVolume) map[string]buildapi.BuildStrategyVolume {
	res := make(map[string]buildapi.BuildStrategyVolume)

	for _, v := range expected {
		res[v.Name] = v
	}

	return res
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"strings"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

var systemReservedParamKeys = map[string]bool{
	"BUILDER_IMAGE": true,
	"DOCKERFILE":    true,
	"CONTEXT_DIR":   true,
}

// overrideParams allows to override an existing list of parameters with a second list,
// as long as their entry names matches
func overrideParams(originalParams []buildv1alpha1.ParamValue, overrideParams []buildv1alpha1.ParamValue) []buildv1alpha1.ParamValue {
	if len(overrideParams) == 0 {
		return originalParams
	}

	if len(originalParams) == 0 && len(overrideParams) > 0 {
		return overrideParams
	}

	// Build a map from originalParams
	originalMap := make(map[string]buildv1alpha1.ParamValue)
	for _, p := range originalParams {
		originalMap[p.Name] = p
	}

	// Override param in map where the name matches in both originalParams and overrideParams.
	// Extend map to add parameters that only the overrideParams list contains.
	for _, p := range overrideParams {
		originalMap[p.Name] = p
	}

	// drop results on a slice and return
	paramsList := []buildv1alpha1.ParamValue{}

	for k := range originalMap {
		paramsList = append(paramsList, originalMap[k])
	}

	return paramsList
}

// IsSystemReservedParameter verifies if we are using a system reserved parameter name
func IsSystemReservedParameter(param string) bool {
	return systemReservedParamKeys[param] || strings.HasPrefix(param, "shp-")
}

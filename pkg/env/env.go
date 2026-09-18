// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
)

var (
	forbiddenEnvVarNames    = map[string]bool{}
	forbiddenEnvVarPrefixes []string
)

// SetForbiddenEnvVars replaces the forbidden environment variable blocklist.
// Entries ending with "*" are treated as prefix matches; all others are exact matches.
func SetForbiddenEnvVars(entries []string) {
	names := make(map[string]bool)
	var prefixes []string
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if strings.HasSuffix(e, "*") {
			prefixes = append(prefixes, strings.TrimSuffix(e, "*"))
		} else {
			names[e] = true
		}
	}
	forbiddenEnvVarNames = names
	forbiddenEnvVarPrefixes = prefixes
}

func IsForbiddenEnvVar(name string) bool {
	if forbiddenEnvVarNames[name] {
		return true
	}
	for _, prefix := range forbiddenEnvVarPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// MergeEnvVars merges two slices of corev1.EnvVar into a new slice of corev1.EnvVar that is then returned
// if overwriteValues is false, this function will return an error if a duplicate EnvVar name is encountered
// if overwriteValues is true, this function will overwrite the existing value with the new value if a duplicate is encountered
func MergeEnvVars(env1 []corev1.EnvVar, env2 []corev1.EnvVar, overwriteValues bool) ([]corev1.EnvVar, error) {
	if len(env1) == 0 && len(env2) == 0 {
		return []corev1.EnvVar{}, nil
	}

	// create a map of the env2 variables with the name as the key and
	// their index as the value so we can do value replacements later if overwriteValues is true
	envIndices := make(map[string]int)

	// errs holds a slice of error objects from the merge process
	var errs []error

	merged := make([]corev1.EnvVar, 0, len(env2)+len(env1))

	for _, envVar := range env2 {
		index, exists := envIndices[envVar.Name]

		switch {
		case exists && overwriteValues:
			merged[index] = envVar
		case exists && !overwriteValues:
			errs = append(errs, fmt.Errorf("environment variable %q already exists", envVar.Name))
		default:
			envIndices[envVar.Name] = len(merged)
			merged = append(merged, envVar)
		}
	}

	// merge the env1 variables into the result following a few simple rules
	// based on if the name already exists and whether overwriteValues is true or false
	for _, envVar := range env1 {
		if IsForbiddenEnvVar(envVar.Name) {
			errs = append(errs, fmt.Errorf("environment variable %q is forbidden for security reasons", envVar.Name))
			continue
		}

		index, exists := envIndices[envVar.Name]

		switch {
		case exists && overwriteValues:
			merged[index] = envVar
		case exists && !overwriteValues:
			errs = append(errs, fmt.Errorf("environment variable %q already exists", envVar.Name))
		default:
			envIndices[envVar.Name] = len(merged)
			merged = append(merged, envVar)
		}
	}

	// kerrors.NewAggregate will return nil if the slice is empty
	// or an aggregated error otherwise
	return merged, kerrors.NewAggregate(errs)
}

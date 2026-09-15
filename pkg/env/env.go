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

// MergeEnvVars merges one slice of corev1.EnvVar into another slice of corev1.EnvVar
// if overwriteValues is false, this function will return an error if a duplicate EnvVar name is encountered
// if overwriteValues is true, this function will overwrite the existing value with the new value if a duplicate is encountered
func MergeEnvVars(from []corev1.EnvVar, into []corev1.EnvVar, overwriteValues bool) ([]corev1.EnvVar, error) {
	if len(from) == 0 && len(into) == 0 {
		return []corev1.EnvVar{}, nil
	}

	// create a map of the original (into) env vars with the name as the key and
	// their index as the value so we can do value replacements later if overwriteValues is true
	envIndices := make(map[string]int)

	// errs holds a slice of error objects from the merge process
	var errs []error

	merged := make([]corev1.EnvVar, 0, len(into)+len(from))

	for _, o := range into {
		index, exists := envIndices[o.Name]

		switch {
		case exists && overwriteValues:
			merged[index] = o
		case exists && !overwriteValues:
			errs = append(errs, fmt.Errorf("environment variable %q already exists", o.Name))
		default:
			envIndices[o.Name] = len(merged)
			merged = append(merged, o)
		}
	}

	// merge the new env vars into the original env vars list following a few simple rules
	// based on if the name already exists and whether overwriteValues is true or false
	for _, n := range from {
		if IsForbiddenEnvVar(n.Name) {
			errs = append(errs, fmt.Errorf("environment variable %q is forbidden for security reasons", n.Name))
			continue
		}

		index, exists := envIndices[n.Name]

		switch {
		case exists && overwriteValues:
			merged[index] = n
		case exists && !overwriteValues:
			errs = append(errs, fmt.Errorf("environment variable %q already exists", n.Name))
		default:
			envIndices[n.Name] = len(merged)
			merged = append(merged, n)
		}
	}

	// kerrors.NewAggregate will return nil if the slice is empty
	// or an aggregated error otherwise
	return merged, kerrors.NewAggregate(errs)
}

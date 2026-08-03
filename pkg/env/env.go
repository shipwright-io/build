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
	// if from, into, or both are empty, there is no need to run through the processing logic
	// just quickly return the appropriate value
	if len(from) == 0 && len(into) == 0 {
		return []corev1.EnvVar{}, nil
	} else if len(from) == 0 {
		return into, nil
	} else if len(into) == 0 {
		return from, nil
	}

	// create a map of the original (into) env vars with the name as the key and
	// their index as the value so we can do value replacements later if overwriteValues is true
	originalEnvs := make(map[string]int)

	for i, o := range into {
		originalEnvs[o.Name] = i
	}

	// errs holds a slice of error objects from the merge process
	var errs []error

	// merge the new env vars into the original env vars list following a few simple rules
	// based on if the name already exists and whether overwriteValues is true or false
	for _, n := range from {
		if IsForbiddenEnvVar(n.Name) {
			errs = append(errs, fmt.Errorf("environment variable %q is forbidden for security reasons", n.Name))
			continue
		}

		_, exists := originalEnvs[n.Name]

		switch {
		case exists && overwriteValues:
			into[originalEnvs[n.Name]] = n
		case exists && !overwriteValues:
			errs = append(errs, fmt.Errorf("environment variable %q already exists", n.Name))
		default:
			into = append(into, n)
		}
	}

	// kerrors.NewAggregate will return nil if the slice is empty
	// or an aggregated error otherwise
	return into, kerrors.NewAggregate(errs)
}

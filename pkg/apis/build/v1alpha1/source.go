// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

// Source describes the Git source repository to fetch.
type Source struct {
	// URL describes the URL of the Git repository.
	URL string `json:"url"`

	// Revision describes the Git revision (e.g., branch, tag, commit SHA,
	// etc.) to fetch.
	//
	// If not defined, it will fallback to the repository's default branch.
	//
	// +optional
	Revision *string `json:"revision,omitempty"`

	// ContextDir is a path to subfolder in the repo. Optional.
	//
	// +optional
	ContextDir *string `json:"contextDir,omitempty"`

	// Credentials references a Secret that contains credentials to access
	// the repository.
	//
	// +optional
	Credentials *corev1.LocalObjectReference `json:"credentials,omitempty"`
}

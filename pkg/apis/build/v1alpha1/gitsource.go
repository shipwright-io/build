// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

// GitSource contains the versioned source code metadata
// This is similar to OpenShift BuildConfig Git Source API
type GitSource struct {

	// URL of the git repo
	URL string `json:"url"`

	// Ref is a git reference. Optional. If not defined, it will fallback to the git repository default branch.
	Revision *string `json:"revision,omitempty"`

	// ContextDir is a path to subfolder in the repo. Optional.
	// +optional
	ContextDir *string `json:"contextDir,omitempty"`

	// SecretRef refers to the secret that contains credentials to access the git repo. Optional.
	SecretRef *corev1.LocalObjectReference `json:"credentials,omitempty"`
}

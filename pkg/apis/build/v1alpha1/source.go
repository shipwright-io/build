// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

// PruneOption defines the supported options for image pruning
type PruneOption string

const (
	// Do not delete image after it was pulled
	PruneNever PruneOption = "Never"

	// Delete image after it was successfully pulled
	PruneAfterPull PruneOption = "AfterPull"
)

// BundleContainer describes the source code bundle container to pull
type BundleContainer struct {
	// Image reference, i.e. quay.io/org/image:tag
	Image string `json:"image"`

	// Prune specifies whether the image is suppose to be deleted. Allowed
	// values are 'Never' (no deletion) and `AfterPull` (removal after the
	// image was successfully pulled from the registry).
	//
	// If not defined, it defaults to 'Never'.
	//
	// +optional
	Prune *PruneOption `json:"prune,omitempty"`
}

// Source describes the Git source repository to fetch.
type Source struct {
	// URL describes the URL of the Git repository.
	//
	// +optional
	URL *string `json:"url,omitempty"`

	// BundleContainer
	//
	// +optional
	BundleContainer *BundleContainer `json:"bundleContainer,omitempty"`

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

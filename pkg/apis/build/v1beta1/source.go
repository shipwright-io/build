// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// PruneOption defines the supported options for image pruning
type PruneOption string

// BuildSourceType enumerates build source type names.
type BuildSourceType string

// LocalType represents a alternative build workflow that instead of `git clone` the repository, it
// employs the data uploaded by the user, streamed directly into the POD.
const LocalType BuildSourceType = "Local"

// GitType represents the default build workflow behaviour. Where the source code is `git clone` from
// a public or private repository
const GitType BuildSourceType = "Git"

// OCIArtifactType represents a source code bundle container image to pull. This is where the source code resides.
const OCIArtifactType BuildSourceType = "OCI"

const (
	// Do not delete image after it was pulled
	PruneNever PruneOption = "Never"

	// Delete image after it was successfully pulled
	PruneAfterPull PruneOption = "AfterPull"
)

type Local struct {
	// Timeout how long the BuildSource execution must take.
	//
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Name of the local step
	Name string `json:"name,omitempty"`
}

// Git describes the git repository to pull
type Git struct {
	// URL describes the URL of the Git repository.
	URL string `json:"url"`

	// Revision describes the Git revision (e.g., branch, tag, commit SHA,
	// etc.) to fetch.
	//
	// If not defined, it will fallback to the repository's default branch.
	//
	// +optional
	Revision *string `json:"revision,omitempty"`

	// CloneSecret references a Secret that contains credentials to access
	// the repository.
	//
	// +optional
	CloneSecret *string `json:"cloneSecret,omitempty"`
}

// OCIArtifact describes the source code bundle container to pull
type OCIArtifact struct {
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

	// PullSecret references a Secret that contains credentials to access
	// the repository.
	//
	// +optional
	PullSecret *string `json:"pullSecret,omitempty"`
}

// Source describes the build source type to fetch.
type Source struct {
	// Type is the BuildSource qualifier, the type of the source.
	Type BuildSourceType `json:"type"`

	// ContextDir is a path to subfolder in the repo. Optional.
	//
	// +optional
	ContextDir *string `json:"contextDir,omitempty"`

	// OCIArtifact contains the details for the source of type OCIArtifact
	//
	// +optional
	OCIArtifact *OCIArtifact `json:"ociArtifact,omitempty"`

	// Git contains the details for the source of type Git
	//
	// +optional
	Git *Git `json:"git,omitempty"`

	// Local contains the details for the source of type Local
	//
	// +optional
	Local *Local `json:"local,omitempty"`
}

// BuildRunSource describes the local source to use
type BuildRunSource struct {
	// Type is the BuildRunSource qualifier, the type of the source.
	// Only Local is supported.
	//
	Type BuildSourceType `json:"type"`

	// Local contains the details for the source of type Local
	//
	// +optional
	Local *Local `json:"local,omitempty"`
}

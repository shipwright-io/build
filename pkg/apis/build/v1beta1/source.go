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

// OCIArtifactType represents a build whose source code is in a "scratch" container image, also known as an OCI artifact.
const OCIArtifactType BuildSourceType = "OCI"

const (
	// Do not delete image after it was pulled
	PruneNever PruneOption = "Never"

	// Delete image after it was successfully pulled
	PruneAfterPull PruneOption = "AfterPull"
)

// Local describes how to obtain source code streamed in from a remote machine's local directory.
// Local source code can be streamed into a build using the shp command line.
type Local struct {
	// Timeout is the maximum duration the build should wait for source code to be streamed in from
	// a remote machine's local directory.
	//
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Name of the local step
	Name string `json:"name,omitempty"`
}

// Git describes how to obtain source code from a git repository.
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

	// Depth specifies the depth of the shallow clone.
	// If not specified the default is set to 1.
	// Values greater than 1 will create a clone with the specified depth.
	// If value is 0, it will create a full git history clone.
	//
	// +optional
	Depth *int `json:"depth,omitempty"`
}

// OCIArtifact describes how to obtain source code from a container image, also known as an OCI
// artifact.
type OCIArtifact struct {
	// Image is a reference to a container image to be pulled from a container registry.
	// For example, quay.io/org/image:tag
	Image string `json:"image"`

	// Prune specifies whether the image containing the source code should be deleted.
	// Allowed values are 'Never' (no deletion) and `AfterPull` (removal after the
	// image was successfully pulled from the registry).
	//
	// If not defined, it defaults to 'Never'.
	//
	// +optional
	Prune *PruneOption `json:"prune,omitempty"`

	// PullSecret references a Secret that contains credentials to access
	// the container image.
	//
	// +optional
	PullSecret *string `json:"pullSecret,omitempty"`
}

// Source describes the source code to fetch for the build.
type Source struct {
	// Type is the type of source code used as input for the build. Allowed values are
	// `Git`, `OCI`, and `Local`.
	Type BuildSourceType `json:"type"`

	// ContextDir is a path to a subdirectory within the source code that should be used as the
	// build root directory. Optional.
	//
	// +optional
	ContextDir *string `json:"contextDir,omitempty"`

	// OCIArtifact contains the details for obtaining source code from a container image, also
	// known as an OCI artifact.
	//
	// +optional
	OCIArtifact *OCIArtifact `json:"ociArtifact,omitempty"`

	// Git contains the details for obtaining source code from a git repository.
	//
	// +optional
	Git *Git `json:"git,omitempty"`

	// Local contains the details for obtaining source code that is streamed in from a remote
	// machine's local directory.
	//
	// +optional
	Local *Local `json:"local,omitempty"`
}

// BuildRunSource describes the source to use in a BuildRun, overriding the value of the parent
// Build object.
type BuildRunSource struct {
	// Type is the BuildRunSource qualifier, the type of the source.
	// Only `Local` is supported.
	//
	Type BuildSourceType `json:"type"`

	// Local contains the details for the source of type Local
	//
	// +optional
	Local *Local `json:"local,omitempty"`
}

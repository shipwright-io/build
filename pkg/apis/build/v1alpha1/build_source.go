// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildSourceType enumerates build source type names.
type BuildSourceType string

// LocalCopy represents a alternative build workflow that instead of `git clone` the repository, it
// employs the data uploaded by the user, streamed directly into the POD.
const LocalCopy BuildSourceType = "LocalCopy"

// HTTP defines a (HTTP) remote artifact, which will be downloaded into the build POD, right before
// the build process starts. Represents a remote dependency.
//
// NOTICE: HTTP artifact downloads are deprecated. This feature will be removed in a future release.
const HTTP BuildSourceType = "HTTP"

// BuildSource remote artifact definition, also known as "sources". Simple "name" and "url" pairs,
// initially without "credentials" (authentication) support yet.
type BuildSource struct {
	// Name instance entry.
	Name string `json:"name"`

	// Type is the BuildSource qualifier, the type of the data-source.
	//
	// +optional
	Type BuildSourceType `json:"type,omitempty"`

	// Timeout how long the BuildSource execution must take.
	//
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// URL remote artifact location.
	//
	// +optional
	URL string `json:"url,omitempty"`
}

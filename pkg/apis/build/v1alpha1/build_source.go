// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildSourceType enumerates build source type names.
type BuildSourceType string

// LocalCopy defines a source type that waits for user input, as in a local copy into a POD.
const LocalCopy BuildSourceType = "LocalCopy"

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

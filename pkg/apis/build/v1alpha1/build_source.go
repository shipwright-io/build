// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

// BuildSourceType enumerates build source type names.
type BuildSourceType string

// BuildSource remote artifact definition, also known as "sources". Simple "name" and "url" pairs,
// initially without "credentials" (authentication) support yet.
type BuildSource struct {
	// Name instance entry.
	Name string `json:"name"`

	// URL remote artifact location.
	URL string `json:"url"`
}

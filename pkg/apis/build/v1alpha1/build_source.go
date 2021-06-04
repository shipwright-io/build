// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import corev1 "k8s.io/api/core/v1"

// BuildSourceType enumerates build source type names.
type BuildSourceType string

const (
	BuildSourceTypeGit  = "Git"
	BuildSourceTypeHTTP = "HTTP"
)

// BuildSource remote artifact definition, also known as "sources". Simple "name" and "url" pairs,
// initially without "credentials" (authentication) support yet.
type BuildSource struct {
	// Name instance entry.
	Name string `json:"name"`

	// Type is the type of build source. Must be one of the following:
	// - "Git"
	// - "HTTP"
	// TODO - populate this field on admission with a webhook
	//
	// +optional
	Type BuildSourceType `json:"type,omitempty"`

	// Destination is an optional sub-directory where the source code can be downloaded.
	// If not specified, will default to the working directory of the build.
	//
	// +optional
	Destination string `json:"destination,omitempty"`

	// Git contains the information needed to obtain source code via git-clone.
	Git *GitBuildSource `json:"git,omitempty"`

	// HTTP contains the information needed to obtain source code via HTTP download.
	HTTP *HTTPBuildSource `json:"http,omitempty"`

	// URL remote artifact location. DEPRECATED - use http.url instead
	URL string `json:"url"`
}

type GitBuildSource struct {
	// URL describes the URL of the Git repository.
	URL string `json:"url"`

	// Revision describes the Git revision (e.g., branch, tag, commit SHA,
	// etc.) to fetch.
	//
	// If not defined, it will fallback to the repository's default branch.
	//
	// +optional
	Revision string `json:"revision,omitempty"`

	// Credentials references a Secret that contains credentials to access
	// the repository.
	//
	// +optional
	Credentials *corev1.LocalObjectReference `json:"credentials,omitempty"`
}

type HTTPBuildSource struct {
	// URL is the location of a source file to be downloaded via HTTP.
	URL string `json:"url"`
}

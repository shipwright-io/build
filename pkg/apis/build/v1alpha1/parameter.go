// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

// Parameter defines the data structure that would be used for
// expressing arbitrary key/value pairs for the execution of a build
type Parameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

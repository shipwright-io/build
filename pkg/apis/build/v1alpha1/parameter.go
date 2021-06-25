// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

// ParamValue is a key/value that populates a strategy parameter
// used in the execution of the strategy steps
type ParamValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

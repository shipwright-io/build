// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

type ObjectKeyRef struct {

	// Name of the object
	// +required
	Name string `json:"name"`

	// Key inside the object
	// +required
	Key string `json:"key"`

	// An optional format to add pre- or suffix to the object value. For example 'KEY=${SECRET_VALUE}' or 'KEY=${CONFIGMAP_VALUE}' depending on the context.
	// +optional
	Format *string `json:"format"`
}

// The value type contains the properties for a value, this allows for an
// easy extension in the future to support more kinds
type SingleValue struct {

	// The value of the parameter
	// +optional
	Value *string `json:"value"`

	// The ConfigMap value of the parameter
	// +optional
	ConfigMapValue *ObjectKeyRef `json:"configMapValue"`

	// The secret value of the parameter
	// +optional
	SecretValue *ObjectKeyRef `json:"secretValue"`
}

// ParamValue is a key/value that populates a strategy parameter
// used in the execution of the strategy steps
type ParamValue struct {

	// Inline the properties of a value
	// +optional
	*SingleValue `json:",inline"`

	// Name of the parameter
	// +required
	Name string `json:"name"`

	// Values of an array parameter
	// +optional
	Values []SingleValue `json:"values,omitempty"`
}

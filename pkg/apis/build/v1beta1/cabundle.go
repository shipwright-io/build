// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package v1beta1

// CABundle is a set of sources whose data will be added to system trust bundle.
// +structType=atomic
// +kubebuilder:validation:ExactlyOneOf=configMap;secret
type CABundle struct {
	// configMap is a reference (by name) to a ConfigMap's `data` key(s), or to a
	// list of ConfigMap's `data` key(s) using label selector, in the namespace.
	// +optional
	ConfigMap *SourceObjectKeySelector `json:"configMap,omitempty"`

	// secret is a reference (by name) to a Secret's `data` key(s), or to a
	// list of Secret's `data` key(s) using label selector, in the namespace.
	// +optional
	Secret *SourceObjectKeySelector `json:"secret,omitempty"`
}

// SourceObjectKeySelector is a reference to a source object and its `data` key(s)
// in the trust namespace.
// +structType=atomic
type SourceObjectKeySelector struct {
	// Name is the name of the source object in the trust namespace.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name,omitempty"`

	// Key of the entry in the object's `data` field to be used.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Key string `json:"key,omitempty"`
}

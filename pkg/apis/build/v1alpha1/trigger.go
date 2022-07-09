// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

// Trigger represents the webhook trigger configuration for a Build.
type Trigger struct {
	// When the list of scenarios when a new build should take place.
	When []TriggerWhen `json:"when,omitempty"`

	// SecretRef points to a local object carrying the secret token to validate webhook request.
	//
	// +optional
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`
}

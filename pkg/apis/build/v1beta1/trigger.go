// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package v1beta1

// Trigger represents the webhook trigger configuration for a Build.
type Trigger struct {
	// When the list of scenarios when a new build should take place.
	When []TriggerWhen `json:"when,omitempty"`

	// TriggerSecret points to a local object carrying the secret token to validate webhook request.
	//
	// +optional
	TriggerSecret *string `json:"triggerSecret,omitempty"`
}

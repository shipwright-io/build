// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package v1alpha1

// TriggerType set of TriggerWhen valid names.
type TriggerType string

const (
	// GitHubWebHookTrigger GitHubWebHookTrigger trigger type name.
	GitHubWebHookTrigger TriggerType = "GitHub"

	// ImageTrigger Image trigger type name.
	ImageTrigger TriggerType = "Image"

	// PipelineTrigger Tekton Pipeline trigger type name.
	PipelineTrigger TriggerType = "Pipeline"
)

// GitHubEventName set of WhenGitHub valid event names.
type GitHubEventName string

const (
	// GitHubPullRequestEvent github pull-request event name.
	GitHubPullRequestEvent GitHubEventName = "PullRequest"

	// GitHubPushEvent git push webhook event name.
	GitHubPushEvent GitHubEventName = "Push"
)

// WhenImage attributes to match Image events.
type WhenImage struct {
	// Names fully qualified image names.
	//
	// +optional
	Names []string `json:"names,omitempty"`
}

// WhenGitHub attributes to match GitHub events.
type WhenGitHub struct {
	// Events GitHub event names.
	//
	// +kubebuilder:validation:MinItems=1
	Events []GitHubEventName `json:"events,omitempty"`

	// Branches slice of branch names where the event applies.
	//
	// +optional
	Branches []string `json:"branches,omitempty"`
}

// WhenObjectRef attributes to reference local Kubernetes objects.
type WhenObjectRef struct {
	// Name target object name.
	//
	// +optional
	Name string `json:"name,omitempty"`

	// Status object status.
	Status []string `json:"status,omitempty"`

	// Selector label selector.
	//
	// +optional
	Selector map[string]string `json:"selector,omitempty"`
}

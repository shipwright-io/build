// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package v1alpha1

// TriggerWhen a given scenario where the webhook trigger is applicable.
type TriggerWhen struct {
	// Name name or the short description of the trigger condition.
	Name string `json:"name"`

	// Type the event type
	Type TriggerType `json:"type"`

	// GitHub describes how to trigger builds based on GitHub (SCM) events.
	//
	// +optional
	GitHub *WhenGitHub `json:"github,omitempty"`

	// Image slice of image names where the event applies.
	//
	// +optional
	Image *WhenImage `json:"image,omitempty"`

	// ObjectRef describes how to match a foreign resource, either using the name or the label
	// selector, plus the current resource status.
	//
	// +optional
	ObjectRef *WhenObjectRef `json:"objectRef,omitempty"`
}

// GetBranches return a slice of branch names based on the WhenTypeName informed.
func (w *TriggerWhen) GetBranches(whenType TriggerType) []string {
	switch whenType {
	case GitHubWebHookTrigger:
		if w.GitHub == nil {
			return nil
		}
		return w.GitHub.Branches
	}
	return nil
}

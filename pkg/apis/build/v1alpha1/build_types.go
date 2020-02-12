package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildSpec defines the desired state of Build
type BuildSpec struct {
	Source GitSource `json:"source"`

	StrategyRef string `json:"strategy"`

	// +optional
	BuilderImage *string `json:"builderImage,omitempty"`

	// +optional
	Dockerfile *string `json:"dockerfile,omitempty"`

	// +optional
	PathContext *string `json:"pathContext,omniempty"`

	// +optional
	Parameters *Parameter `json:"parameters,omitempty"`

	OutputImage string `json:"outputImage"`
}

// BuildStatus defines the observed state of Build
type BuildStatus struct {
	Status string `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Build is the Schema for the builds API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=builds,scope=Namespaced
type Build struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildSpec   `json:"spec,omitempty"`
	Status BuildStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildList contains a list of Build
type BuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Build `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Build{}, &BuildList{})
}

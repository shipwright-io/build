package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildSpec defines the desired state of Build
type BuildSpec struct {

	// Source refers to the Git repository constaining the
	// source code to be built.
	Source GitSource `json:"source"`

	// StrategyRef refers to the BuildStrategy to be used to
	// build the container image.
	// Note: Using metav1.ObjectMeta  instead of corev1.LocalObjectReference
	// because the BuildStrategy may or may not be in the same namespace.
	StrategyRef metav1.ObjectMeta `json:"strategy"`

	// BuilderImage refers to the image containing the build tools
	// inside which the source code would be built.
	// +optional
	BuilderImage *string `json:"builderImage,omitempty"`

	// Dockerfile is the path to the Dockerfile to be used for
	// build strategies which bank on the Dockerfile for building
	// an image.
	// +optional
	Dockerfile *string `json:"dockerfile,omitempty"`

	// Parameters contains name-value that could be used to loosely
	// type parameters in the BuildStrategy.
	// +optional
	Parameters *[]Parameter `json:"parameters,omitempty"`

	// Output refers to the location where the generated
	// image would be pushed to.
	Output Output `json:"output"`
}

// Output refers to the location where the generated
// image would be pushed to.
type Output struct {

	// ImageURL is the URL where the image will be pushed to.
	ImageURL string `json:"image"`

	// SecretRef is a reference to the Secret containing the
	// credentials to push the image to the registry
	// +optional
	SecretRef *corev1.LocalObjectReference `json:"credentials,omitempty"`
}

// BuildStatus defines the observed state of Build
type BuildStatus struct {
	Status string `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Build is the Schema for the builds API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=builds,scope=Namespaced
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status",description="The status of this Build"
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

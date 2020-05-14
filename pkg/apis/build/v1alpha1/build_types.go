package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	LabelBuild           = "build.build.dev/name"
	LabelBuildGeneration = "build.build.dev/generation"
)

// BuildSpec defines the desired state of Build
type BuildSpec struct {

	// Source refers to the Git repository containing the
	// source code to be built.
	Source GitSource `json:"source"`

	// StrategyRef refers to the BuildStrategy to be used to
	// build the container image.
	// There are namespaced scope and cluster scope BuildStrategy
	StrategyRef *StrategyRef `json:"strategy"`

	// BuilderImage refers to the image containing the build tools
	// inside which the source code would be built.
	// +optional
	BuilderImage *Image `json:"builder,omitempty"`

	// Dockerfile is the path to the Dockerfile to be used for
	// build strategies which bank on the Dockerfile for building
	// an image.
	// +optional
	Dockerfile *string `json:"dockerfile,omitempty"`

	// Parameters contains name-value that could be used to loosely
	// type parameters in the BuildStrategy.
	// +optional
	Parameters *[]Parameter `json:"parameters,omitempty"`

	// Compute Resources required by the build container.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// Output refers to the location where the generated
	// image would be pushed to.
	Output Image `json:"output"`

	// Timeout defines the maximum run time of a build run.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

// Image refers to an container image with credentials
type Image struct {

	// ImageURL is the URL where the image will be pushed to.
	ImageURL string `json:"image"`

	// SecretRef is a reference to the Secret containing the
	// credentials to push the image to the registry
	// +optional
	SecretRef *corev1.LocalObjectReference `json:"credentials,omitempty"`
}

// BuildStatus defines the observed state of Build
type BuildStatus struct {
	// The Register status of the Build
	// +optional
	Registered corev1.ConditionStatus `json:"registered,omitempty"`

	// The reason of the registered Build, either an error or succeed message
	// +optional
	Reason string `json:"reason,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Build is the Schema for the builds API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=builds,scope=Namespaced
// +kubebuilder:printcolumn:name="Registered",type="string",JSONPath=".status.registered",description="The register status of the Build"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.reason",description="The reason of the registered Build, either an error or succeed message"
// +kubebuilder:printcolumn:name="BuildStrategyKind",type="string",JSONPath=".spec.strategy.kind",description="The BuildStrategy type which is used for this Build"
// +kubebuilder:printcolumn:name="BuildStrategyName",type="string",JSONPath=".spec.strategy.name",description="The BuildStrategy name which is used for this Build"
// +kubebuilder:printcolumn:name="CreationTime",type="date",JSONPath=".metadata.creationTimestamp",description="The create time of this Build"
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

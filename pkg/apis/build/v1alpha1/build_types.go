// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// LabelBuild is a label key for defining the build name
	LabelBuild = "build.build.dev/name"

	// LabelBuildGeneration is a label key for defining the build generation
	LabelBuildGeneration = "build.build.dev/generation"

	// AnnotationBuildRunDeletion is a label key for enabling/disabling the BuildRun deletion
	AnnotationBuildRunDeletion = "build.build.dev/build-run-deletion"
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

	// Runtime represents the runtime-image
	// +optional
	Runtime *Runtime `json:"runtime,omitempty"`

	// Output refers to the location where the generated
	// image would be pushed to.
	Output Image `json:"output"`

	// Timeout defines the maximum run time of a build run.
	// +optional
	// +kubebuilder:validation:Format=duration
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

// Runtime represents the runtime-image, created using parts of builder-image, and a different
// base-image than originally.
type Runtime struct {
	// Base runtime base image.
	// +optional
	Base Image `json:"base,omitempty"`

	// Env environment variables for runtime.
	// +optional
	Env map[string]string `json:"env,omitempty"`

	// Labels map of additional labels to be applied on image.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// WorkDir runtime image working directory `WORKDIR`.
	// +optional
	WorkDir string `json:"workDir,omitempty"`

	// Run arbitrary commands to run before copying data into runtime-image.
	// +optional
	Run []string `json:"run,omitempty"`

	// Paths list of directories/files to be copied into runtime-image, using colon ":" to split up source and destination paths.
	// +optional
	Paths []string `json:"paths,omitempty"`

	// User definitions of user and group for runtime-image.
	User *User `json:"user,omitempty"`

	// Entrypoint runtime-image entrypoint.
	// +optional
	Entrypoint []string `json:"entrypoint,omitempty"`
}

// User holds the user name and group information for runtime-image.
type User struct {
	// Name user name to be employed in runtime-image.
	Name string `json:"name"`

	// Group group name or GID employed in runtime-image.
	// +optional
	Group string `json:"group,omitempty"`
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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Build is the Schema representing a Build definition
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

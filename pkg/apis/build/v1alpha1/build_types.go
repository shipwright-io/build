// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildReason is a type used for populating the
// Build Status.Reason field
type BuildReason string

const (
	// SucceedStatus indicates that all validations Succeeded
	SucceedStatus BuildReason = "Succeeded"
	// BuildStrategyNotFound indicates that a namespaced-scope strategy was not found in the namespace
	BuildStrategyNotFound BuildReason = "BuildStrategyNotFound"
	// ClusterBuildStrategyNotFound indicates that a cluster-scope strategy was not found
	ClusterBuildStrategyNotFound BuildReason = "ClusterBuildStrategyNotFound"
	// SetOwnerReferenceFailed indicates that setting ownerReferences between a Build and a BuildRun failed
	SetOwnerReferenceFailed BuildReason = "SetOwnerReferenceFailed"
	// SpecSourceSecretRefNotFound indicates the referenced secret in source is missing
	SpecSourceSecretRefNotFound BuildReason = "SpecSourceSecretRefNotFound"
	// SpecOutputSecretRefNotFound indicates the referenced secret in output is missing
	SpecOutputSecretRefNotFound BuildReason = "SpecOutputSecretRefNotFound"
	// SpecBuilderSecretRefNotFound indicates the referenced secret in builder is missing
	SpecBuilderSecretRefNotFound BuildReason = "SpecBuilderSecretRefNotFound"
	// MultipleSecretRefNotFound indicates that multiple secrets are missing
	MultipleSecretRefNotFound BuildReason = "MultipleSecretRefNotFound"
	// RuntimePathsCanNotBeEmpty indicates that the spec.runtime feature is used but the paths were not specified
	RuntimePathsCanNotBeEmpty BuildReason = "RuntimePathsCanNotBeEmpty"
	// RemoteRepositoryUnreachable indicates the referenced repository is unreachable
	RemoteRepositoryUnreachable BuildReason = "RemoteRepositoryUnreachable"
	// AllValidationsSucceeded indicates a Build was successfully validated
	AllValidationsSucceeded = "all validations succeeded"
)

const (
	// BuildDomain is the domain used for all labels and annotations for this resource
	BuildDomain = "build.shipwright.io"

	// LabelBuild is a label key for defining the build name
	LabelBuild = BuildDomain + "/name"

	// LabelBuildGeneration is a label key for defining the build generation
	LabelBuildGeneration = BuildDomain + "/generation"

	// AnnotationBuildRunDeletion is a label key for enabling/disabling the BuildRun deletion
	AnnotationBuildRunDeletion = BuildDomain + "/build-run-deletion"

	// AnnotationBuildRefSecret is an annotation that tells the Build Controller to reconcile on
	// events of the secret only if is referenced by a Build in the same namespace
	AnnotationBuildRefSecret = BuildDomain + "/referenced.secret"

	// AnnotationBuildVerifyRepository tells the Build Controller to check a remote repository. If the annotation is not set
	// or has a value of 'true', the controller triggers the validation. A value of 'false' means the controller
	// will bypass checking the remote repository.
	AnnotationBuildVerifyRepository = BuildDomain + "/verify.repository"
)

// BuildSpec defines the desired state of Build
type BuildSpec struct {
	// Source refers to the Git repository containing the
	// source code to be built.
	Source Source `json:"source"`

	// Strategy references the BuildStrategy to use to build the container
	// image.
	Strategy *Strategy `json:"strategy"`

	// Builder refers to the image containing the build tools inside which
	// the source code would be built.
	//
	// +optional
	Builder *Image `json:"builder,omitempty"`

	// Dockerfile is the path to the Dockerfile to be used for
	// build strategies which bank on the Dockerfile for building
	// an image.
	//
	// +optional
	Dockerfile *string `json:"dockerfile,omitempty"`

	// Parameters contains name-value that could be used to loosely
	// type parameters in the BuildStrategy.
	//
	// +optional
	Parameters *[]Parameter `json:"parameters,omitempty"`

	// Runtime represents the runtime-image.
	//
	// +optional
	Runtime *Runtime `json:"runtime,omitempty"`

	// Output refers to the location where the built image would be pushed.
	Output Image `json:"output"`

	// Timeout defines the maximum amount of time the Build should take to execute.
	//
	// +optional
	// +kubebuilder:validation:Format=duration
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

// StrategyName returns the name of the configured strategy, or 'undefined' in
// case the strategy is nil (not set)
func (buildSpec *BuildSpec) StrategyName() string {
	if buildSpec.Strategy == nil {
		return "undefined (nil strategy)"
	}

	return buildSpec.Strategy.Name
}

// Image refers to an container image with credentials
type Image struct {
	// Image is the reference of the image.
	Image string `json:"image"`

	// Credentials references a Secret that contains credentials to access
	// the image registry.
	//
	// +optional
	Credentials *corev1.LocalObjectReference `json:"credentials,omitempty"`
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

	// The reason of the registered Build, it's an one-word camelcase
	// +optional
	Reason BuildReason `json:"reason,omitempty"`

	// The message of the registered Build, either an error or succeed message
	// +optional
	Message string `json:"message,omitempty"`
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

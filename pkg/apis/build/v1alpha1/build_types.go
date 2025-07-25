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
	// SpecEnvNameCanNotBeBlank indicates that the name for an environment variable is blank
	SpecEnvNameCanNotBeBlank BuildReason = "SpecEnvNameCanNotBeBlank"
	// SpecEnvOnlyOneOfValueOrValueFromMustBeSpecified indicates that both value and valueFrom were specified
	SpecEnvOnlyOneOfValueOrValueFromMustBeSpecified BuildReason = "SpecEnvOnlyOneOfValueOrValueFromMustBeSpecified"
	// RuntimePathsCanNotBeEmpty indicates that the spec.runtime feature is used but the paths were not specified
	RuntimePathsCanNotBeEmpty BuildReason = "RuntimePathsCanNotBeEmpty"
	// RestrictedParametersInUse indicates the definition of reserved shipwright parameters
	RestrictedParametersInUse BuildReason = "RestrictedParametersInUse"
	// WrongParameterValueType indicates that a single value was provided for an array parameter, or vice-versa
	WrongParameterValueType BuildReason = "WrongParameterValueType"
	// UndefinedParameter indicates the definition of param that was not defined in the strategy parameters
	UndefinedParameter BuildReason = "UndefinedParameter"
	// InconsistentParameterValues indicates that parameter values have more than one of configMapValue, secretValue, or value set
	InconsistentParameterValues BuildReason = "InconsistentParameterValues"
	// EmptyArrayItemParameterValues indicates that array parameters contain an item where none of configMapValue, secretValue, or value is set
	EmptyArrayItemParameterValues BuildReason = "EmptyArrayItemParameterValues"
	// IncompleteConfigMapValueParameterValues indicates that a configMapValue is specified where the name or the key is empty
	IncompleteConfigMapValueParameterValues BuildReason = "IncompleteConfigMapValueParameterValues"
	// IncompleteSecretValueParameterValues indicates that a secretValue is specified where the name or the key is empty
	IncompleteSecretValueParameterValues BuildReason = "IncompleteSecretValueParameterValues"
	// RemoteRepositoryUnreachable indicates the referenced repository is unreachable
	RemoteRepositoryUnreachable BuildReason = "RemoteRepositoryUnreachable"
	// BuildNameInvalid indicates the build name is invalid
	BuildNameInvalid BuildReason = "BuildNameInvalid"
	// VolumeDoesNotExist indicates that volume referenced by the Build does not exist, therefore Build cannot be run
	VolumeDoesNotExist BuildReason = "VolumeDoesNotExist"
	// VolumeNotOverridable indicates that volume defined by build is not set as overridable in the strategy
	VolumeNotOverridable BuildReason = "VolumeNotOverridable"
	// UndefinedVolume indicates that volume defined by build is not found in the strategy
	UndefinedVolume BuildReason = "UndefinedVolume"
	// TriggerNameCanNotBeBlank indicates the trigger condition does not have a name
	TriggerNameCanNotBeBlank BuildReason = "TriggerNameCanNotBeBlank"
	// TriggerInvalidType indicates the trigger type is invalid
	TriggerInvalidType BuildReason = "TriggerInvalidType"
	// TriggerInvalidGitHubWebHook indicates the trigger type GitHub is invalid
	TriggerInvalidGitHubWebHook BuildReason = "TriggerInvalidGitHubWebHook"
	// TriggerInvalidImage indicates the trigger type Image is invalid
	TriggerInvalidImage BuildReason = "TriggerInvalidImage"
	// TriggerInvalidPipeline indicates the trigger type Pipeline is invalid
	TriggerInvalidPipeline BuildReason = "TriggerInvalidPipeline"

	// AllValidationsSucceeded indicates a Build was successfully validated
	AllValidationsSucceeded = "all validations succeeded"
)

// BuildReasonPtr returns a pointer to the passed BuildReason.
func BuildReasonPtr(s BuildReason) *BuildReason {
	return &s
}

// ConditionStatusPtr returns a pointer to the passed ConditionStatus.
func ConditionStatusPtr(s corev1.ConditionStatus) *corev1.ConditionStatus {
	return &s
}

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

const (
	// OutputImageZeroTimestamp indicates that the UNIX timestamp 0 is to be used
	OutputImageZeroTimestamp = "Zero"

	// OutputImageSourceTimestamp indicates that the timestamp of the respective source it to be used
	OutputImageSourceTimestamp = "SourceTimestamp"

	// OutputImageBuildTimestamp indicates that the current timestamp of the build run itself is to be used
	OutputImageBuildTimestamp = "BuildTimestamp"
)

// BuildSpec defines the desired state of Build
type BuildSpec struct {
	// Source refers to the Git repository containing the
	// source code to be built.
	Source Source `json:"source"`

	// Sources slice of BuildSource, defining external build artifacts complementary to VCS
	// (`.spec.source`) data.
	//
	// +optional
	//
	// NOTICE: Multiple sources in a build are deprecated. This feature will be removed in a future release.
	Sources []BuildSource `json:"sources,omitempty"`

	// Trigger defines the scenarios where a new build should be triggered.
	//
	// +optional
	Trigger *Trigger `json:"trigger,omitempty"`

	// Strategy references the BuildStrategy to use to build the container
	// image.
	Strategy Strategy `json:"strategy"`

	// Builder refers to the image containing the build tools inside which
	// the source code would be built.
	//
	// NOTICE: Builder is deprecated, and will be removed in a future release.
	// Build strategies which rely on "builder" should provide an equivalent parameter instead.
	//
	// +optional
	Builder *Image `json:"builder,omitempty"`

	// Dockerfile is the path to the Dockerfile to be used for
	// build strategies which bank on the Dockerfile for building
	// an image.
	//
	// NOTICE: Dockerfile is deprecated, and will be removed in a future release.
	// Build strategies which rely on "dockerfile" should provide an equivalent parameter instead.
	//
	// +optional
	Dockerfile *string `json:"dockerfile,omitempty"`

	// Params is a list of key/value that could be used
	// to set strategy parameters
	//
	// +optional
	ParamValues []ParamValue `json:"paramValues,omitempty"`

	// Output refers to the location where the built image would be pushed.
	Output Image `json:"output"`

	// Timeout defines the maximum amount of time the Build should take to execute.
	//
	// +optional
	// +kubebuilder:validation:Format=duration
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Env contains additional environment variables that should be passed to the build container
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Contains information about retention params
	//
	// +optional
	Retention *BuildRetention `json:"retention,omitempty"`

	// Volumes contains volume Overrides of the BuildStrategy volumes in case those are allowed
	// to be overridden. Must only contain volumes that exist in the corresponding BuildStrategy
	// +optional
	Volumes []BuildVolume `json:"volumes,omitempty"`
}

// BuildVolume is a volume that will be mounted in build pod during build step
type BuildVolume struct {
	// Name of the Build Volume
	// +required
	Name string `json:"name"`

	// Description of the Build Volume
	//
	// NOTICE: Description is deprecated, and will be removed in a future release.
	//
	// +optional
	Description *string `json:"description,omitempty"`

	// Represents the source of a volume to mount
	// +optional
	corev1.VolumeSource `json:",inline"`
}

// StrategyName returns the name of the configured strategy, or 'undefined' in
// case the strategy is nil (not set)
func (buildSpec *BuildSpec) StrategyName() string {
	if buildSpec == nil {
		return "undefined (nil buildSpec)"
	}

	return buildSpec.Strategy.Name
}

// Image refers to an container image with credentials
type Image struct {
	// Image is the reference of the image.
	Image string `json:"image"`

	// Insecure defines whether the registry is not secure
	//
	// +optional
	Insecure *bool `json:"insecure,omitempty"`

	// Credentials references a Secret that contains credentials to access
	// the image registry.
	//
	// +optional
	Credentials *corev1.LocalObjectReference `json:"credentials,omitempty"`

	// Annotations references the additional annotations to be applied on the image
	//
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels references the additional labels to be applied on the image
	//
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Timestamp references the optional image timestamp to be set, valid values are:
	// - "Zero", to set 00:00:00 UTC on 1 January 1970
	// - "SourceTimestamp", to set the source timestamp dereived from the input source
	// - "BuildTimestamp", to set the timestamp of the current build itself
	// - Parsable integer number defined as the epoch seconds
	// - or nil/empty to not set any specific timestamp
	//
	// +optional
	Timestamp *string `json:"timestamp,omitempty"`
}

// BuildStatus defines the observed state of Build
//
// NOTICE: This is deprecated and will be removed in a future release.
type BuildStatus struct {
	// The Register status of the Build
	// +optional
	Registered *corev1.ConditionStatus `json:"registered,omitempty"`

	// The reason of the registered Build, it's an one-word camelcase
	// +optional
	Reason *BuildReason `json:"reason,omitempty"`

	// The message of the registered Build, either an error or succeed message
	// +optional
	Message *string `json:"message,omitempty"`
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

	Spec   BuildSpec   `json:"spec"`
	Status BuildStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildList contains a list of Build
type BuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Build `json:"items"`
}

// BuildRetention struct for buildrun cleanup
type BuildRetention struct {
	// FailedLimit defines the maximum number of failed buildruns that should exist.
	//
	// +optional
	// +kubebuilder:validation:Minimum=1
	FailedLimit *uint `json:"failedLimit,omitempty"`
	// SucceededLimit defines the maximum number of succeeded buildruns that should exist.
	//
	// +optional
	// +kubebuilder:validation:Minimum=1
	SucceededLimit *uint `json:"succeededLimit,omitempty"`
	// TTLAfterFailed defines the maximum duration of time the failed buildrun should exist.
	//
	// +optional
	// +kubebuilder:validation:Format=duration
	TTLAfterFailed *metav1.Duration `json:"ttlAfterFailed,omitempty"`
	// TTLAfterSucceeded defines the maximum duration of time the succeeded buildrun should exist.
	//
	// +optional
	// +kubebuilder:validation:Format=duration
	TTLAfterSucceeded *metav1.Duration `json:"ttlAfterSucceeded,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Build{}, &BuildList{})
}

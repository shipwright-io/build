// Copyright The Shipwright Contributors
// 
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildStrategySpec defines the desired state of BuildStrategy
type BuildStrategySpec struct {
	BuildSteps []BuildStep `json:"buildSteps,omitempty"`
}

// BuildStep defines a partial step that needs to run in container for
// building the image.
type BuildStep struct {
	corev1.Container `json:",inline"`
}

// BuildStrategyStatus defines the observed state of BuildStrategy
type BuildStrategyStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildStrategy is the Schema representing a strategy in the namespace scope to build images from source code.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=buildstrategies,scope=Namespaced,shortName=bs;bss
type BuildStrategy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildStrategySpec   `json:"spec,omitempty"`
	Status BuildStrategyStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildStrategyList contains a list of BuildStrategy
type BuildStrategyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BuildStrategy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BuildStrategy{}, &BuildStrategyList{})
}

// StrategyRef can be used to refer to a specific instance of a buildstrategy.
// Copied from CrossVersionObjectReference: https://github.com/kubernetes/kubernetes/blob/169df7434155cbbc22f1532cba8e0a9588e29ad8/pkg/apis/autoscaling/types.go#L64
type StrategyRef struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name"`
	// BuildStrategyKind indicates the kind of the buildstrategy, namespaced or cluster scoped.
	Kind *BuildStrategyKind `json:"kind,omitempty"`
	// API version of the referent
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
}

// Check that Build may be validated and defaulted.
// BuildStrategyKind defines the type of BuildStrategy used by the build.
type BuildStrategyKind string

const (
	// NamespacedBuildStrategyKind indicates that the buildstrategy type has a namespaced scope.
	NamespacedBuildStrategyKind BuildStrategyKind = "BuildStrategy"
	// ClusterBuildStrategyKind indicates that buildstrategy type has a cluster scope.
	ClusterBuildStrategyKind BuildStrategyKind = "ClusterBuildStrategy"
)

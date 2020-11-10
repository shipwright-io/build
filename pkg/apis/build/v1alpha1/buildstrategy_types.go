// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// LabelBuildStrategyName is a label key for defining the build strategy name
	LabelBuildStrategyName = "buildstrategy.build.dev/name"

	// LabelBuildStrategyGeneration is a label key for defining the build strategy generation
	LabelBuildStrategyGeneration = "buildstrategy.build.dev/generation"
)

// +genclient
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

// GetName returns the name of the build strategy
func (s BuildStrategy) GetName() string {
	return s.Name
}

// GetGeneration returns the current generation sequence number of the build
// strategy resource
func (s BuildStrategy) GetGeneration() int64 {
	return s.Generation
}

// GetResourceLabels returns labels that define the build strategy name and
// generation to be used in labels map of a resource
func (s BuildStrategy) GetResourceLabels() map[string]string {
	return map[string]string{
		LabelBuildStrategyName:       s.Name,
		LabelBuildStrategyGeneration: strconv.FormatInt(s.Generation, 10),
	}
}

// GetBuildSteps returns the spec build steps of the build strategy
func (s BuildStrategy) GetBuildSteps() []BuildStep {
	return s.Spec.BuildSteps
}

func init() {
	SchemeBuilder.Register(&BuildStrategy{}, &BuildStrategyList{})
}

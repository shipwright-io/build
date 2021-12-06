// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ClusterBuildStrategyDomain is the domain used for all labels and annotations for this resource
	ClusterBuildStrategyDomain = "clusterbuildstrategy.shipwright.io"

	// LabelClusterBuildStrategyName is a label key for defining the cluster build strategy name
	LabelClusterBuildStrategyName = ClusterBuildStrategyDomain + "/name"

	// LabelClusterBuildStrategyGeneration is a label key for defining the cluster build strategy generation
	LabelClusterBuildStrategyGeneration = ClusterBuildStrategyDomain + "/generation"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterBuildStrategy is the Schema representing a strategy in the cluster scope to build images from source code.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=clusterbuildstrategies,scope=Cluster,shortName=cbs;cbss
type ClusterBuildStrategy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildStrategySpec   `json:"spec,omitempty"`
	Status BuildStrategyStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterBuildStrategyList contains a list of ClusterBuildStrategy
type ClusterBuildStrategyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterBuildStrategy `json:"items"`
}

// GetAnnotations returns the annotations of the build strategy
func (s ClusterBuildStrategy) GetAnnotations() map[string]string {
	return s.Annotations
}

// GetName returns the name of the build strategy
func (s ClusterBuildStrategy) GetName() string {
	return s.Name
}

// GetGeneration returns the current generation sequence number of the build
// strategy resource
func (s ClusterBuildStrategy) GetGeneration() int64 {
	return s.Generation
}

// GetResourceLabels returns labels that define the build strategy name and
// generation to be used in labels map of a resource
func (s ClusterBuildStrategy) GetResourceLabels() map[string]string {
	return map[string]string{
		LabelClusterBuildStrategyName:       s.Name,
		LabelClusterBuildStrategyGeneration: strconv.FormatInt(s.Generation, 10),
	}
}

// GetBuildSteps returns the spec build steps of the build strategy
func (s ClusterBuildStrategy) GetBuildSteps() []BuildStep {
	return s.Spec.BuildSteps
}

// GetParameters returns the parameters defined by the build strategy
func (s ClusterBuildStrategy) GetParameters() []Parameter {
	return s.Spec.Parameters
}

func init() {
	SchemeBuilder.Register(&ClusterBuildStrategy{}, &ClusterBuildStrategyList{})
}

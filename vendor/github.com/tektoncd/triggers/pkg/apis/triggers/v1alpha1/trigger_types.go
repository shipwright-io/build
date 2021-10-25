/*
Copyright 2020 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"encoding/json"
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

// TriggerSpec represents a connection between TriggerSpecBinding,
// and TriggerSpecTemplate; TriggerSpecBinding provides extracted values for
// TriggerSpecTemplate to then create resources from.
type TriggerSpec struct {
	Bindings []*TriggerSpecBinding `json:"bindings"`
	Template TriggerSpecTemplate   `json:"template"`
	// +optional
	Name         string                `json:"name,omitempty"`
	Interceptors []*TriggerInterceptor `json:"interceptors,omitempty"`
	// ServiceAccountName optionally associates credentials with each trigger;
	// Unlike EventListeners, this should be scoped to the same namespace
	// as the Trigger itself
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

type TriggerSpecTemplate struct {
	Ref        *string              `json:"ref,omitempty"`
	APIVersion string               `json:"apiversion,omitempty"`
	Spec       *TriggerTemplateSpec `json:"spec,omitempty"`
}

type TriggerSpecBinding struct {
	// Name is the name of the binding param
	// Mutually exclusive with Ref
	Name string `json:"name,omitempty"`
	// Value is the value of the binding param. Can contain JSONPath
	// Has to be pointer since "" is a valid value
	// Required if Name is also specified.
	Value *string `json:"value,omitempty"`

	// Ref is a reference to a TriggerBinding kind.
	// Mutually exclusive with Name
	Ref string `json:"ref,omitempty"`

	// Kind can only be provided if Ref is also provided. Defaults to TriggerBinding
	Kind TriggerBindingKind `json:"kind,omitempty"`

	// APIVersion of the binding ref
	APIVersion string `json:"apiversion,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Trigger defines a mapping of an input event to parameters. This is used
// to extract information from events to be passed to TriggerTemplates within a
// Trigger.
// +k8s:openapi-gen=true
type Trigger struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec holds the desired state of the Trigger
	// +optional
	Spec TriggerSpec `json:"spec"`
}

// TriggerInterceptor provides a hook to intercept and pre-process events
type TriggerInterceptor struct {
	// Optional name to identify the current interceptor configuration
	Name *string `json:"name,omitempty"`
	// Ref refers to the Interceptor to use
	Ref InterceptorRef `json:"ref"`
	// Params are the params to send to the interceptor
	Params []InterceptorParams `json:"params,omitempty"`

	// WebhookInterceptor refers to an old style webhook interceptor service
	Webhook *WebhookInterceptor `json:"webhook,omitempty"`

	// Deprecated old fields below
	DeprecatedGitHub    *GitHubInterceptor    `json:"github,omitempty"`
	DeprecatedGitLab    *GitLabInterceptor    `json:"gitlab,omitempty"`
	DeprecatedCEL       *CELInterceptor       `json:"cel,omitempty"`
	DeprecatedBitbucket *BitbucketInterceptor `json:"bitbucket,omitempty"`
}

// InterceptorParams defines a key-value pair that can be passed on an interceptor
type InterceptorParams struct {
	Name  string               `json:"name"`
	Value apiextensionsv1.JSON `json:"value"`
}

// InterceptorRef provides a Reference to a ClusterInterceptor
type InterceptorRef struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name,omitempty"`
	// InterceptorKind indicates the kind of the Interceptor, namespaced or cluster scoped.
	// Currently only InterceptorKind is ClusterInterceptor, so the only valid value
	// is the default one
	// +optional
	Kind InterceptorKind `json:"kind,omitempty"`
	// API version of the referent
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
}

// InterceptorKind defines the type of Interceptor used by the Trigger.
type InterceptorKind string

const (
	// ClusterTaskKind indicates that task type has a cluster scope.
	ClusterInterceptorKind InterceptorKind = "ClusterInterceptor"
)

func (ti *TriggerInterceptor) defaultInterceptorKind() {
	if ti.Ref.Kind == "" {
		ti.Ref.Kind = ClusterInterceptorKind
	}
}

func (ti *TriggerInterceptor) updateCoreInterceptors() error {
	if ti == nil {
		return nil
	}
	if ti.Ref.Name != "" {
		return nil
	}
	ti.Ref.Name = ti.GetName()
	ti.Params = []InterceptorParams{}
	switch ti.Ref.Name {
	case "bitbucket":
		if err := addToParams(&ti.Params, "secretRef", ti.DeprecatedBitbucket.SecretRef); err != nil {
			return err
		}
		if err := addToParams(&ti.Params, "eventTypes", ti.DeprecatedBitbucket.EventTypes); err != nil {
			return err
		}
		ti.DeprecatedBitbucket = nil
	case "gitlab":
		if err := addToParams(&ti.Params, "secretRef", ti.DeprecatedGitLab.SecretRef); err != nil {
			return err
		}
		if err := addToParams(&ti.Params, "eventTypes", ti.DeprecatedGitLab.EventTypes); err != nil {
			return err
		}
		ti.DeprecatedGitLab = nil
	case "github":
		if err := addToParams(&ti.Params, "secretRef", ti.DeprecatedGitHub.SecretRef); err != nil {
			return err
		}
		if err := addToParams(&ti.Params, "eventTypes", ti.DeprecatedGitHub.EventTypes); err != nil {
			return err
		}
		ti.DeprecatedGitHub = nil
	case "cel":
		if err := addToParams(&ti.Params, "filter", ti.DeprecatedCEL.Filter); err != nil {
			return err
		}
		if err := addToParams(&ti.Params, "overlays", ti.DeprecatedCEL.Overlays); err != nil {
			return err
		}
		ti.DeprecatedCEL = nil
	}
	return nil
}

func addToParams(params *[]InterceptorParams, name string, val interface{}) error {
	if val == nil {
		return nil
	}
	v, err := toV1JSON(val)
	if err != nil {
		return err
	}
	*params = append(*params, InterceptorParams{
		Name:  name,
		Value: v,
	})
	return nil
}

func toV1JSON(v interface{}) (apiextensionsv1.JSON, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return apiextensionsv1.JSON{}, fmt.Errorf("json.Marshal() failed: %s", err)
	}
	return apiextensionsv1.JSON{
		Raw: b,
	}, nil
}

// GetName returns the name for the given interceptor
func (ti *TriggerInterceptor) GetName() string {
	// This is temporary until we implement #869
	name := ""
	switch {
	case ti.Ref.Name != "":
		name = ti.Ref.Name
	case ti.DeprecatedBitbucket != nil:
		name = "bitbucket"
	case ti.DeprecatedCEL != nil:
		name = "cel"
	case ti.DeprecatedGitHub != nil:
		name = "github"
	case ti.DeprecatedGitLab != nil:
		name = "gitlab"
	}
	return name
}

// WebhookInterceptor provides a webhook to intercept and pre-process events
type WebhookInterceptor struct {
	// ObjectRef is a reference to an object that will resolve to a cluster DNS
	// name to use as the EventInterceptor. Either objectRef or url can be specified
	// +optional
	ObjectRef *corev1.ObjectReference `json:"objectRef,omitempty"`
	// +optional
	URL *apis.URL `json:"url,omitempty"`
	// Header is a group of key-value pairs that can be appended to the
	// interceptor request headers. This allows the interceptor to make
	// decisions specific to an EventListenerTrigger.
	Header []v1beta1.Param `json:"header,omitempty"`
}

// BitbucketInterceptor provides a webhook to intercept and pre-process events
type BitbucketInterceptor struct {
	SecretRef  *SecretRef `json:"secretRef,omitempty"`
	EventTypes []string   `json:"eventTypes,omitempty"`
}

// GitHubInterceptor provides a webhook to intercept and pre-process events
type GitHubInterceptor struct {
	SecretRef  *SecretRef `json:"secretRef,omitempty"`
	EventTypes []string   `json:"eventTypes,omitempty"`
}

// GitLabInterceptor provides a webhook to intercept and pre-process events
type GitLabInterceptor struct {
	SecretRef  *SecretRef `json:"secretRef,omitempty"`
	EventTypes []string   `json:"eventTypes,omitempty"`
}

// CELInterceptor provides a webhook to intercept and pre-process events
type CELInterceptor struct {
	Filter   string       `json:"filter,omitempty"`
	Overlays []CELOverlay `json:"overlays,omitempty"`
}

// CELOverlay provides a way to modify the request body using DeprecatedCEL expressions
type CELOverlay struct {
	Key        string `json:"key,omitempty"`
	Expression string `json:"expression,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TriggerList contains a list of Triggers.
// We don't use this but it's required for certain codegen features.
type TriggerList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Trigger `json:"items"`
}

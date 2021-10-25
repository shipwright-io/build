/*
Copyright 2021 The Tekton Authors

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
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// Check that EventListener may be validated and defaulted.
var _ apis.Validatable = (*ClusterInterceptor)(nil)
var _ apis.Defaultable = (*ClusterInterceptor)(nil)

// +genclient
// +genclient:nonNamespaced
// +genreconciler:krshapedlogic=false
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// ClusterInterceptor describes a pluggable interceptor including configuration
// such as the fields it accepts and its deployment address. The type is based on
// the Validating/MutatingWebhookConfiguration types for configuring AdmissionWebhooks
type ClusterInterceptor struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterInterceptorSpec `json:"spec"`
	// +optional
	Status ClusterInterceptorStatus `json:"status"`
}

// ClusterInterceptorSpec describes the Spec for an ClusterInterceptor
type ClusterInterceptorSpec struct {
	ClientConfig ClientConfig `json:"clientConfig"`
}

// ClusterInterceptorStatus holds the status of the ClusterInterceptor
// +k8s:deepcopy-gen=true
type ClusterInterceptorStatus struct {
	duckv1.Status `json:",inline"`

	// ClusterInterceptor is Addressable and exposes the URL where the Interceptor is running
	duckv1.AddressStatus `json:",inline"`
}

// ClientConfig describes how a client can communicate with the Interceptor
type ClientConfig struct {
	// URL is a fully formed URL pointing to the interceptor
	// Mutually exclusive with Service
	URL *apis.URL `json:"url,omitempty"`

	// Service is a reference to a Service object where the interceptor is running
	// Mutually exclusive with URL
	Service *ServiceReference `json:"service,omitempty"`
}

var defaultPort = int32(80)

// ServiceReference is a reference to a Service object
// with an optional path
type ServiceReference struct {
	// Name is the name of the service
	Name string `json:"name"`

	// Namespace is the namespace of the service
	Namespace string `json:"namespace"`

	// Path is an optional URL path
	// +optional
	Path string `json:"path,omitempty"`

	// Port is a valid port number
	Port *int32 `json:"port,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ClusterInterceptorList contains a list of ClusterInterceptor
// We don't use this but it's required for certain codegen features.
type ClusterInterceptorList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterInterceptor `json:"items"`
}

var ErrNilURL = errors.New("interceptor URL was nil")

// ResolveAddress returns the URL where the interceptor is running using its clientConfig
func (it *ClusterInterceptor) ResolveAddress() (*apis.URL, error) {
	if url := it.Spec.ClientConfig.URL; url != nil {
		return url, nil
	}
	svc := it.Spec.ClientConfig.Service
	if svc == nil {
		return nil, ErrNilURL
	}
	port := defaultPort
	if svc.Port != nil {
		port = *svc.Port
	}
	url := &apis.URL{
		Scheme: "http", // TODO: Support HTTPs if caBundle is present
		Host:   fmt.Sprintf("%s.%s.svc:%d", svc.Name, svc.Namespace, port),
		Path:   svc.Path,
	}
	return url, nil
}

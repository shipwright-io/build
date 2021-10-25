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
	"context"

	"knative.dev/pkg/apis"
)

// Validate ClusterInterceptor
func (it *ClusterInterceptor) Validate(ctx context.Context) *apis.FieldError {
	if apis.IsInDelete(ctx) {
		return nil
	}
	return it.Spec.validate(ctx)
}

func (s *ClusterInterceptorSpec) validate(ctx context.Context) (errs *apis.FieldError) {
	if s.ClientConfig.URL != nil && s.ClientConfig.Service != nil {
		errs = errs.Also(apis.ErrMultipleOneOf("spec.clientConfig.url", "spec.clientConfig.service"))
	}
	if svc := s.ClientConfig.Service; svc != nil {
		if svc.Namespace == "" {
			errs = errs.Also(apis.ErrMissingField("spec.clientConfig.service.namespace"))
		}
		if svc.Name == "" {
			errs = errs.Also(apis.ErrMissingField("spec.clientConfig.service.name"))
		}
	}
	return errs
}

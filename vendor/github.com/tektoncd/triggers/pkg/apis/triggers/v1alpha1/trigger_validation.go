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
	"context"
	"fmt"
	"net/http"

	"github.com/google/cel-go/cel"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/validate"
	"knative.dev/pkg/apis"
)

// Validate validates a Trigger
func (t *Trigger) Validate(ctx context.Context) *apis.FieldError {
	errs := validate.ObjectMetadata(t.GetObjectMeta()).ViaField("metadata")
	if apis.IsInDelete(ctx) {
		return nil
	}
	return errs.Also(t.Spec.validate(ctx).ViaField("spec"))
}

func (t *TriggerSpec) validate(ctx context.Context) *apis.FieldError {
	// Validate optional Bindings
	errs := triggerSpecBindingArray(t.Bindings).validate(ctx)
	// Validate required TriggerTemplate
	errs = errs.Also(t.Template.validate(ctx))

	// Validate optional Interceptors
	for i, interceptor := range t.Interceptors {
		errs = errs.Also(interceptor.validate(ctx).ViaField(fmt.Sprintf("interceptors[%d]", i)))
	}

	return errs
}

func (t TriggerSpecTemplate) validate(ctx context.Context) (errs *apis.FieldError) {
	// Optional explicit match
	if t.APIVersion != "" {
		if t.APIVersion != "v1alpha1" {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Errorf("invalid apiVersion"), "template.apiVersion"))
		}
	}

	switch {
	case t.Spec != nil && t.Ref != nil:
		errs = errs.Also(apis.ErrMultipleOneOf("template.spec", "template.ref"))
	case t.Spec == nil && t.Ref == nil:
		errs = errs.Also(apis.ErrMissingOneOf("template.spec", "template.ref"))
	case t.Spec != nil:
		errs = errs.Also(t.Spec.validate(ctx))
	case t.Ref == nil || *t.Ref == "":
		errs = errs.Also(apis.ErrMissingField("template.ref"))
	}
	return errs
}

func (t triggerSpecBindingArray) validate(ctx context.Context) (errs *apis.FieldError) {
	for i, b := range t {
		switch {
		case b.Ref != "":
			switch {
			case b.Name != "": // Cannot specify both Ref and Name
				errs = errs.Also(apis.ErrMultipleOneOf(fmt.Sprintf("bindings[%d].Ref", i), fmt.Sprintf("bindings[%d].Name", i)))
			case b.Kind != NamespacedTriggerBindingKind && b.Kind != ClusterTriggerBindingKind: // Kind must be valid
				errs = errs.Also(apis.ErrInvalidValue(fmt.Errorf("invalid kind"), fmt.Sprintf("bindings[%d].kind", i)))
			}
		case b.Name != "":
			if b.Value == nil { // Value is mandatory if Name is specified
				errs = errs.Also(apis.ErrMissingField(fmt.Sprintf("bindings[%d].Value", i)))
			}
		default:
			errs = errs.Also(apis.ErrMissingOneOf(fmt.Sprintf("bindings[%d].Ref", i), fmt.Sprintf("bindings[%d].Spec", i), fmt.Sprintf("bindings[%d].Name", i)))
		}
	}
	return errs
}

func (i *TriggerInterceptor) validate(ctx context.Context) (errs *apis.FieldError) {
	if i.Webhook == nil && i.DeprecatedGitHub == nil && i.DeprecatedGitLab == nil && i.DeprecatedCEL == nil && i.DeprecatedBitbucket == nil {
		if i.Ref.Name == "" { // Check to see if Interceptor referenced using Ref
			errs = errs.Also(apis.ErrMissingField("interceptor"))
		}
	}

	// Enforce oneof
	numSet := 0
	if i.Webhook != nil {
		numSet++
	}
	if i.DeprecatedGitHub != nil {
		numSet++
	}
	if i.DeprecatedGitLab != nil {
		numSet++
	}
	if i.DeprecatedBitbucket != nil {
		numSet++
	}

	if numSet > 1 {
		errs = errs.Also(apis.ErrMultipleOneOf("interceptor.webhook", "interceptor.github", "interceptor.gitlab"))
	}

	if i.Webhook != nil {
		if i.Webhook.ObjectRef == nil || i.Webhook.ObjectRef.Name == "" {
			errs = errs.Also(apis.ErrMissingField("interceptor.webhook.objectRef"))
		}
		w := i.Webhook
		if w.ObjectRef.Kind != "Service" {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Errorf("invalid kind"), "interceptor.webhook.objectRef.kind"))
		}

		// Optional explicit match
		if w.ObjectRef.APIVersion != "v1" {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Errorf("invalid apiVersion"), "interceptor.webhook.objectRef.apiVersion"))
		}

		for i, header := range w.Header {
			// Enforce non-empty canonical header keys
			if len(header.Name) == 0 || http.CanonicalHeaderKey(header.Name) != header.Name {
				errs = errs.Also(apis.ErrInvalidValue(fmt.Errorf("invalid header name"), fmt.Sprintf("interceptor.webhook.header[%d].name", i)))
			}
			// Enforce non-empty header values
			if header.Value.Type == pipelinev1.ParamTypeString {
				if len(header.Value.StringVal) == 0 {
					errs = errs.Also(apis.ErrInvalidValue(fmt.Errorf("invalid header value"), fmt.Sprintf("interceptor.webhook.header[%d].value", i)))
				}
			} else if len(header.Value.ArrayVal) == 0 {
				errs = errs.Also(apis.ErrInvalidValue(fmt.Errorf("invalid header value"), fmt.Sprintf("interceptor.webhook.header[%d].value", i)))
			}
		}
	}

	if i.DeprecatedCEL != nil {
		if i.DeprecatedCEL.Filter == "" && len(i.DeprecatedCEL.Overlays) == 0 {
			errs = errs.Also(apis.ErrMultipleOneOf("cel.filter", "cel.overlays"))
		}
		env, err := cel.NewEnv()
		if err != nil {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Errorf("failed to create a DeprecatedCEL env: %s", err), "cel.filter"))
		}
		if i.DeprecatedCEL.Filter != "" {
			if _, issues := env.Parse(i.DeprecatedCEL.Filter); issues != nil && issues.Err() != nil {
				errs = errs.Also(apis.ErrInvalidValue(fmt.Errorf("failed to parse the DeprecatedCEL filter: %s", issues.Err()), "cel.filter"))
			}
		}
		for _, v := range i.DeprecatedCEL.Overlays {
			if _, issues := env.Parse(v.Expression); issues != nil && issues.Err() != nil {
				errs = errs.Also(apis.ErrInvalidValue(fmt.Errorf("failed to parse the DeprecatedCEL overlay: %s", issues.Err()), "cel.overlay"))
			}
		}
	}
	return errs
}

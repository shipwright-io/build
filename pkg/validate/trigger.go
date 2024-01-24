// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package validate

import (
	"context"
	"fmt"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/utils/pointer"
)

// Trigger implements the interface BuildPath with the objective of applying validations against the
// `.spec.trigger` related attributes.
type Trigger struct {
	build *build.Build // build instance
}

// validate goes through the trigger "when" conditions to validate each entry.
func (t *Trigger) validate(triggerWhen []build.TriggerWhen) []error {
	var allErrs []error
	for _, when := range triggerWhen {
		if when.Name == "" {
			t.build.Status.Reason = build.BuildReasonPtr(build.TriggerNameCanNotBeBlank)
			t.build.Status.Message = pointer.String("name is not set on when trigger condition")
			allErrs = append(allErrs, fmt.Errorf("%s", *t.build.Status.Message))
		}

		switch when.Type {
		case build.GitHubWebHookTrigger:
			if when.GitHub == nil {
				t.build.Status.Reason = build.BuildReasonPtr(build.TriggerInvalidGitHubWebHook)
				t.build.Status.Message = pointer.String(fmt.Sprintf(
					"%q is missing required attribute `.github`", when.Name,
				))
				allErrs = append(allErrs, fmt.Errorf("%s", *t.build.Status.Message))
			} else {
				if len(when.GitHub.Events) == 0 {
					t.build.Status.Reason = build.BuildReasonPtr(build.TriggerInvalidGitHubWebHook)
					t.build.Status.Message = pointer.String(fmt.Sprintf(
						"%q is missing required attribute `.github.events`", when.Name,
					))
					allErrs = append(allErrs, fmt.Errorf("%s", *t.build.Status.Message))
				}
			}
		case build.ImageTrigger:
			if when.Image == nil {
				t.build.Status.Reason = build.BuildReasonPtr(build.TriggerInvalidImage)
				t.build.Status.Message = pointer.String(fmt.Sprintf(
					"%q is missing required attribute `.image`", when.Name,
				))
				allErrs = append(allErrs, fmt.Errorf("%s", *t.build.Status.Message))
			} else {
				if len(when.Image.Names) == 0 {
					t.build.Status.Reason = build.BuildReasonPtr(build.TriggerInvalidImage)
					t.build.Status.Message = pointer.String(fmt.Sprintf(
						"%q is missing required attribute `.image.names`", when.Name,
					))
					allErrs = append(allErrs, fmt.Errorf("%s", *t.build.Status.Message))
				}
			}
		case build.PipelineTrigger:
			if when.ObjectRef == nil {
				t.build.Status.Reason = build.BuildReasonPtr(build.TriggerInvalidPipeline)
				t.build.Status.Message = pointer.String(fmt.Sprintf(
					"%q is missing required attribute `.objectRef`", when.Name,
				))
				allErrs = append(allErrs, fmt.Errorf("%s", *t.build.Status.Message))
			} else {
				if len(when.ObjectRef.Status) == 0 {
					t.build.Status.Reason = build.BuildReasonPtr(build.TriggerInvalidPipeline)
					t.build.Status.Message = pointer.String(fmt.Sprintf(
						"%q is missing required attribute `.objectRef.status`", when.Name,
					))
					allErrs = append(allErrs, fmt.Errorf("%s", *t.build.Status.Message))
				}
				if when.ObjectRef.Name == "" && len(when.ObjectRef.Selector) == 0 {
					t.build.Status.Reason = build.BuildReasonPtr(build.TriggerInvalidPipeline)
					t.build.Status.Message = pointer.String(fmt.Sprintf(
						"%q is missing required attributes `.objectRef.name` or `.objectRef.selector`",
						when.Name,
					))
					allErrs = append(allErrs, fmt.Errorf("%s", *t.build.Status.Message))
				}
				if when.ObjectRef.Name != "" && len(when.ObjectRef.Selector) > 0 {
					t.build.Status.Reason = build.BuildReasonPtr(build.TriggerInvalidPipeline)
					t.build.Status.Message = pointer.String(fmt.Sprintf(
						"%q contains `.objectRef.name` and `.objectRef.selector`, must be only one",
						when.Name,
					))
					allErrs = append(allErrs, fmt.Errorf("%s", *t.build.Status.Message))
				}
			}
		default:
			t.build.Status.Reason = build.BuildReasonPtr(build.TriggerInvalidType)
			t.build.Status.Message = pointer.String(
				fmt.Sprintf("%q contains an invalid type %q", when.Name, when.Type))
			allErrs = append(allErrs, fmt.Errorf("%s", *t.build.Status.Message))
		}
	}
	return allErrs
}

// ValidatePath validates the `.spec.trigger` path.
func (t *Trigger) ValidatePath(_ context.Context) error {
	if t.build.Spec.Trigger == nil || len(t.build.Spec.Trigger.When) == 0 {
		return nil
	}

	if allErrs := t.validate(t.build.Spec.Trigger.When); len(allErrs) != 0 {
		return fmt.Errorf("%s", kerrors.NewAggregate(allErrs).Error())
	}
	return nil
}

// NewTrigger instantiate Trigger validation helper.
func NewTrigger(b *build.Build) *Trigger {
	return &Trigger{build: b}
}

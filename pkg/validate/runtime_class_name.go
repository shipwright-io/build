// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/ptr"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

// RuntimeClassNameRef contains all required fields
// to validate a RuntimeClassName
type RuntimeClassNameRef struct {
	Build *build.Build // build instance for analysis
}

func NewRuntimeClassName(build *build.Build) *RuntimeClassNameRef {
	return &RuntimeClassNameRef{build}
}

// ValidatePath implements BuildPath interface and validates
// that RuntimeClassName values are valid
func (b *RuntimeClassNameRef) ValidatePath(_ context.Context) error {
	ok, reason, msg := BuildRunRuntimeClassName(b.Build.Spec.RuntimeClassName)
	if !ok {
		b.Build.Status.Reason = ptr.To(build.BuildReason(reason))
		b.Build.Status.Message = ptr.To(msg)
	}
	return nil
}

// BuildRunRuntimeClassName is used to validate the runtimeClassName in the BuildRun object
func BuildRunRuntimeClassName(runtimeClassName *string) (bool, string, string) {
	if runtimeClassName != nil {
		if errs := validation.IsDNS1123Subdomain(*runtimeClassName); len(errs) > 0 {
			return false, string(build.RuntimeClassNameNotValid), fmt.Sprintf("RuntimeClassName not valid: %v", strings.Join(errs, ", "))
		}
	}
	return true, "", ""
}

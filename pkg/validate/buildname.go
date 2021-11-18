// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"strings"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/pointer"
)

// BuildNameRef contains all required fields
// to validate a build name
type BuildNameRef struct {
	Build *build.Build // build instance for analysis
}

// ValidatePath implements BuildPath interface and validates
// that build name is a valid label value
func (b *BuildNameRef) ValidatePath(_ context.Context) error {
	if errs := validation.IsValidLabelValue(b.Build.Name); len(errs) > 0 {
		b.Build.Status.Reason = build.BuildReasonPtr(build.BuildNameInvalid)
		b.Build.Status.Message = pointer.StringPtr(strings.Join(errs, ", "))
	}

	return nil
}

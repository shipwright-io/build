// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/ptr"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

// BuildNameRef contains all required fields
// to validate a build name
type BuildNameRef struct {
	Build *build.Build // build instance for analysis
}

func NewBuildName(build *build.Build) *BuildNameRef {
	return &BuildNameRef{build}
}

// ValidatePath implements BuildPath interface and validates
// that build name is a valid label value
func (b *BuildNameRef) ValidatePath(_ context.Context) error {
	if errs := validation.IsValidLabelValue(b.Build.Name); len(errs) > 0 {
		b.Build.Status.Reason = ptr.To[build.BuildReason](build.BuildNameInvalid)
		b.Build.Status.Message = ptr.To(strings.Join(errs, ", "))
	}

	return nil
}

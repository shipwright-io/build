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

// SchedulerNameRef contains all required fields
// to validate a Scheduler name
type SchedulerNameRef struct {
	Build *build.Build // build instance for analysis
}

func NewSchedulerName(build *build.Build) *SchedulerNameRef {
	return &SchedulerNameRef{build}
}

// ValidatePath implements BuildPath interface and validates
// that SchedulerName values are valid
func (b *SchedulerNameRef) ValidatePath(_ context.Context) error {
	ok, reason, msg := BuildRunSchedulerName(b.Build.Spec.SchedulerName)
	if !ok {
		b.Build.Status.Reason = ptr.To(build.BuildReason(reason))
		b.Build.Status.Message = ptr.To(msg)
	}
	return nil
}

// BuildSchedulerName is used to validate the schedulerName in the BuildRun object
func BuildRunSchedulerName(schedulerName *string) (bool, string, string) {
	if schedulerName != nil {
		if errs := validation.IsQualifiedName(*schedulerName); len(errs) > 0 {
			return false, string(build.SchedulerNameNotValid), fmt.Sprintf("Scheduler name not valid: %v", strings.Join(errs, ", "))
		}
	}
	return true, "", ""
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"strconv"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"k8s.io/utils/pointer"
)

// BuildSpecOutputValidator implements validation interface to add validations for `build.spec.output`.
type BuildSpecOutputValidator struct {
	Build *build.Build // build instance for analysis
}

var _ BuildPath = &BuildSpecOutputValidator{}

func (b *BuildSpecOutputValidator) ValidatePath(_ context.Context) error {
	if b.Build.Spec.Output.Timestamp != nil {
		switch *b.Build.Spec.Output.Timestamp {
		case "":
			// no validation required

		case build.OutputImageZeroTimestamp:
			// no validation required

		case build.OutputImageSourceTimestamp:
			// check that there is a source defined that can be used in combination with source timestamp
			if b.isEmptySource() {
				b.Build.Status.Reason = build.BuildReasonPtr(build.OutputTimestampNotSupported)
				b.Build.Status.Message = pointer.String("cannot use SourceTimestamp output image setting with an empty build source")
			}

		case build.OutputImageBuildTimestamp:
			// no validation required

		default:
			// check that value is parsable integer
			if _, err := strconv.ParseInt(*b.Build.Spec.Output.Timestamp, 10, 64); err != nil {
				b.Build.Status.Reason = build.BuildReasonPtr(build.OutputTimestampNotValid)
				b.Build.Status.Message = pointer.String("output timestamp value is invalid, must be Zero, SourceTimestamp, BuildTimestamp, or number")
			}
		}
	}

	return nil
}

func (b *BuildSpecOutputValidator) isEmptySource() bool {
	return b.Build.Spec.Source == nil ||
		b.Build.Spec.Source.Git == nil && b.Build.Spec.Source.OCIArtifact == nil && b.Build.Spec.Source.Local == nil
}

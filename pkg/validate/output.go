// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"strconv"

	"k8s.io/utils/ptr"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

// BuildSpecOutputValidator implements validation interface to add validations for `buildapi.spec.output`.
type BuildSpecOutputValidator struct {
	Build *buildapi.Build // build instance for analysis
}

var _ BuildPath = &BuildSpecOutputValidator{}

func (b *BuildSpecOutputValidator) ValidatePath(_ context.Context) error {
	if b.Build.Spec.Output.Timestamp != nil {
		switch *b.Build.Spec.Output.Timestamp {
		case "":
			// no validation required

		case buildapi.OutputImageZeroTimestamp:
			// no validation required

		case buildapi.OutputImageSourceTimestamp:
			// check that there is a source defined that can be used in combination with source timestamp
			if b.isEmptySource() {
				b.Build.Status.Reason = ptr.To[buildapi.BuildReason](buildapi.OutputTimestampNotSupported)
				b.Build.Status.Message = ptr.To("cannot use SourceTimestamp output image setting with an empty build source")
			}

		case buildapi.OutputImageBuildTimestamp:
			// no validation required

		default:
			// check that value is parsable integer
			if _, err := strconv.ParseInt(*b.Build.Spec.Output.Timestamp, 10, 64); err != nil {
				b.Build.Status.Reason = ptr.To[buildapi.BuildReason](buildapi.OutputTimestampNotValid)
				b.Build.Status.Message = ptr.To("output timestamp value is invalid, must be Zero, SourceTimestamp, BuildTimestamp, or number")
			}
		}
	}

	return nil
}

func (b *BuildSpecOutputValidator) isEmptySource() bool {
	return b.Build.Spec.Source == nil ||
		b.Build.Spec.Source.Git == nil && b.Build.Spec.Source.OCIArtifact == nil && b.Build.Spec.Source.Local == nil
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"
	"strconv"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
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
				b.Build.Status.Reason = ptr.To[build.BuildReason](build.OutputTimestampNotSupported)
				b.Build.Status.Message = ptr.To("cannot use SourceTimestamp output image setting with an empty build source")
			}

		case build.OutputImageBuildTimestamp:
			// no validation required

		default:
			// check that value is parsable integer
			if _, err := strconv.ParseInt(*b.Build.Spec.Output.Timestamp, 10, 64); err != nil {
				b.Build.Status.Reason = ptr.To[build.BuildReason](build.OutputTimestampNotValid)
				b.Build.Status.Message = ptr.To("output timestamp value is invalid, must be Zero, SourceTimestamp, BuildTimestamp, or number")
			}
		}
	}

	if b.Build.Spec.Output.MultiArch != nil {
		b.validateMultiArch()
	}

	return nil
}

func (b *BuildSpecOutputValidator) validateMultiArch() {
	if valid, reason, msg := ValidateMultiArchPlatforms(b.Build.Spec.Output.MultiArch.Platforms); !valid {
		b.Build.Status.Reason = ptr.To[build.BuildReason](reason)
		b.Build.Status.Message = ptr.To(msg)
		return
	}
	if valid, reason, msg := ValidateMultiArchNodeSelector(b.Build.Spec.NodeSelector); !valid {
		b.Build.Status.Reason = ptr.To[build.BuildReason](reason)
		b.Build.Status.Message = ptr.To(msg)
		return
	}
}

func (b *BuildSpecOutputValidator) isEmptySource() bool {
	return b.Build.Spec.Source == nil ||
		b.Build.Spec.Source.Git == nil && b.Build.Spec.Source.OCIArtifact == nil && b.Build.Spec.Source.Local == nil
}

// ValidateMultiArchPlatforms validates the platforms in a multiArch configuration.
// This is used by BuildRun validation which doesn't go through the Build validator.
func ValidateMultiArchPlatforms(platforms []build.ImagePlatform) (bool, build.BuildReason, string) {
	if len(platforms) == 0 {
		return false, build.MultiArchInvalidPlatform, "multiArch.platforms must contain at least one platform entry"
	}
	for i, p := range platforms {
		if p.OS == "" || p.Arch == "" {
			return false, build.MultiArchInvalidPlatform, fmt.Sprintf("multiArch.platforms[%d] must specify both os and arch", i)
		}
	}
	return true, "", ""
}

// ValidateMultiArchNodeSelector checks that nodeSelector does not conflict with multi-arch scheduling.
func ValidateMultiArchNodeSelector(nodeSelector map[string]string) (bool, build.BuildReason, string) {
	if _, ok := nodeSelector[corev1.LabelOSStable]; ok {
		return false, build.MultiArchNodeSelectorConflict, fmt.Sprintf("nodeSelector must not contain %q when multiArch is configured; the build controller manages os/arch scheduling", corev1.LabelOSStable)
	}
	if _, ok := nodeSelector[corev1.LabelArchStable]; ok {
		return false, build.MultiArchNodeSelectorConflict, fmt.Sprintf("nodeSelector must not contain %q when multiArch is configured; the build controller manages os/arch scheduling", corev1.LabelArchStable)
	}
	return true, "", ""
}


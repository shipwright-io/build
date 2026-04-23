// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"

	corev1 "k8s.io/api/core/v1"
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

	if len(b.Build.Spec.Output.Platforms) > 0 {
		b.checkOutputPlatformsFields()
	}

	return nil
}

func (b *BuildSpecOutputValidator) checkOutputPlatformsFields() {
	if valid, reason, msg := ValidatePlatforms(b.Build.Spec.Output.Platforms); !valid {
		b.Build.Status.Reason = ptr.To(buildapi.BuildReason(reason))
		b.Build.Status.Message = ptr.To(msg)
		return
	}
	if valid, reason, msg := ValidateOutputNodeSelector(b.Build.Spec.NodeSelector); !valid {
		b.Build.Status.Reason = ptr.To(buildapi.BuildReason(reason))
		b.Build.Status.Message = ptr.To(msg)
		return
	}
}

func (b *BuildSpecOutputValidator) isEmptySource() bool {
	return b.Build.Spec.Source == nil ||
		b.Build.Spec.Source.Git == nil && b.Build.Spec.Source.OCIArtifact == nil && b.Build.Spec.Source.Local == nil
}

// os/arch must be the same strings as Node labels kubernetes.io/os and kubernetes.io/arch
// (lowercase a–z, 0–9). Pattern rejects typos like "amd-64" or "Linux"
var platformLabelValueRegexp = regexp.MustCompile(`^[a-z0-9]+$`)

func platformLabelValueValid(s string) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	return platformLabelValueRegexp.MatchString(s)
}

// ValidatePlatforms validates spec.output.platforms when non-empty.
func ValidatePlatforms(platforms []buildapi.ImagePlatform) (bool, string, string) {
	if len(platforms) == 0 {
		return false, string(buildapi.InvalidPlatform), "spec.output.platforms must contain at least one platform entry"
	}
	seen := make(map[string]bool, len(platforms))
	for i, p := range platforms {
		if p.OS == "" || p.Arch == "" {
			return false, string(buildapi.InvalidPlatform), fmt.Sprintf("spec.output.platforms[%d] must specify both os and arch", i)
		}
		if !platformLabelValueValid(p.OS) {
			return false, string(buildapi.InvalidPlatform), fmt.Sprintf("spec.output.platforms[%d].os %q: invalid; use the value your Nodes have for label %q", i, p.OS, corev1.LabelOSStable)
		}
		if !platformLabelValueValid(p.Arch) {
			return false, string(buildapi.InvalidPlatform), fmt.Sprintf("spec.output.platforms[%d].arch %q: invalid; use the value your Nodes have for label %q", i, p.Arch, corev1.LabelArchStable)
		}
		key := p.OS + "/" + p.Arch
		if seen[key] {
			return false, string(buildapi.InvalidPlatform), fmt.Sprintf("spec.output.platforms[%d] is a duplicate of %s", i, key)
		}
		seen[key] = true
	}
	return true, "", ""
}

// ValidateOutputNodeSelector checks that nodeSelector does not conflict with output platform scheduling.
func ValidateOutputNodeSelector(nodeSelector map[string]string) (bool, string, string) {
	if _, ok := nodeSelector[corev1.LabelOSStable]; ok {
		return false, string(buildapi.NodeSelectorPlatformConflict), fmt.Sprintf("nodeSelector must not contain %q when spec.output.platforms is set; the build controller manages os/arch scheduling", corev1.LabelOSStable)
	}
	if _, ok := nodeSelector[corev1.LabelArchStable]; ok {
		return false, string(buildapi.NodeSelectorPlatformConflict), fmt.Sprintf("nodeSelector must not contain %q when spec.output.platforms is set; the build controller manages os/arch scheduling", corev1.LabelArchStable)
	}
	return true, "", ""
}

// ValidatePipelineRunExecutor checks that the controller is configured with PipelineRun executor
// mode, which is required for multi-arch builds to orchestrate per-platform PipelineTasks.
func ValidatePipelineRunExecutor(executor string) (bool, string, string) {
	if executor != "PipelineRun" {
		return false, string(buildapi.ExecutorNotPipelineRun), fmt.Sprintf(
			"multi-arch builds require PipelineRun executor mode, current executor mode: %q", executor)
	}
	return true, "", ""
}

// ValidateMultiArchPreflight runs platform, nodeSelector, and executor checks before
// the reconciler lists Nodes. If this returns (true, "", ""), the caller may List Nodes
// and then call ValidateNodeAvailability.
func ValidateMultiArchPreflight(platforms []buildapi.ImagePlatform, nodeSelector map[string]string, executor string) (bool, string, string) {
	if valid, reason, msg := ValidatePlatforms(platforms); !valid {
		return valid, reason, msg
	}
	if valid, reason, msg := ValidateOutputNodeSelector(nodeSelector); !valid {
		return valid, reason, msg
	}
	return ValidatePipelineRunExecutor(executor)
}

// ValidateNodeAvailability checks that, for each requested platform, the cluster has
// at least one node that is Ready, is not unschedulable, and has kubernetes.io/os and
// kubernetes.io/arch labels matching that platform.
func ValidateNodeAvailability(platforms []buildapi.ImagePlatform, nodes []corev1.Node) (bool, string, string) {
	available := availablePlatforms(nodes)

	for _, p := range platforms {
		key := p.OS + "/" + p.Arch
		if !available[key] {
			return false, string(buildapi.NodePlatformNotFound), fmt.Sprintf(
				"no schedulable node found for platform %s", key)
		}
	}
	return true, "", ""
}

// availablePlatforms returns the set of os/arch combinations for which there is
// at least one node that is Ready, not unschedulable, and has stable OS and arch labels.
func availablePlatforms(nodes []corev1.Node) map[string]bool {
	platforms := make(map[string]bool)
	for _, node := range nodes {
		if node.Spec.Unschedulable {
			continue
		}
		ready := slices.ContainsFunc(node.Status.Conditions, func(c corev1.NodeCondition) bool {
			return c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue
		})
		if !ready {
			continue
		}
		os := node.Labels[corev1.LabelOSStable]
		arch := node.Labels[corev1.LabelArchStable]
		if os != "" && arch != "" {
			platforms[os+"/"+arch] = true
		}
	}
	return platforms
}

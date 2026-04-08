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
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		b.checkMultiArchFields()
	}

	return nil
}

func (b *BuildSpecOutputValidator) checkMultiArchFields() {
	if valid, reason, msg := ValidateMultiArchPlatforms(b.Build.Spec.Output.MultiArch.Platforms); !valid {
		b.Build.Status.Reason = ptr.To(build.BuildReason(reason))
		b.Build.Status.Message = ptr.To(msg)
		return
	}
	if valid, reason, msg := ValidateMultiArchNodeSelector(b.Build.Spec.NodeSelector); !valid {
		b.Build.Status.Reason = ptr.To(build.BuildReason(reason))
		b.Build.Status.Message = ptr.To(msg)
		return
	}
}

func (b *BuildSpecOutputValidator) isEmptySource() bool {
	return b.Build.Spec.Source == nil ||
		b.Build.Spec.Source.Git == nil && b.Build.Spec.Source.OCIArtifact == nil && b.Build.Spec.Source.Local == nil
}

// ValidateMultiArchPlatforms validates the platforms in a multiArch configuration.
func ValidateMultiArchPlatforms(platforms []build.ImagePlatform) (bool, string, string) {
	if len(platforms) == 0 {
		return false, string(build.MultiArchInvalidPlatform), "multiArch.platforms must contain at least one platform entry"
	}
	seen := make(map[string]bool, len(platforms))
	for i, p := range platforms {
		if p.OS == "" || p.Arch == "" {
			return false, string(build.MultiArchInvalidPlatform), fmt.Sprintf("multiArch.platforms[%d] must specify both os and arch", i)
		}
		key := p.OS + "/" + p.Arch
		if seen[key] {
			return false, string(build.MultiArchInvalidPlatform), fmt.Sprintf("multiArch.platforms[%d] is a duplicate of %s", i, key)
		}
		seen[key] = true
	}
	return true, "", ""
}

// ValidateMultiArchNodeSelector checks that nodeSelector does not conflict with multi-arch scheduling.
func ValidateMultiArchNodeSelector(nodeSelector map[string]string) (bool, string, string) {
	if _, ok := nodeSelector[corev1.LabelOSStable]; ok {
		return false, string(build.MultiArchNodeSelectorConflict), fmt.Sprintf("nodeSelector must not contain %q when multiArch is configured; the build controller manages os/arch scheduling", corev1.LabelOSStable)
	}
	if _, ok := nodeSelector[corev1.LabelArchStable]; ok {
		return false, string(build.MultiArchNodeSelectorConflict), fmt.Sprintf("nodeSelector must not contain %q when multiArch is configured; the build controller manages os/arch scheduling", corev1.LabelArchStable)
	}
	return true, "", ""
}

// ValidateMultiArchExecutor checks that the controller is configured with PipelineRun executor
// mode, which is required for multi-arch builds to orchestrate per-platform PipelineTasks.
func ValidateMultiArchExecutor(executor string) (bool, string, string) {
	if executor != "PipelineRun" {
		return false, string(build.MultiArchExecutorNotPipelineRun), fmt.Sprintf(
			"multi-arch builds require PipelineRun executor mode, current executor mode: %q", executor)
	}
	return true, "", ""
}

// ValidateMultiArch runs all multi-arch pre-flight checks in order: platform
// validity, nodeSelector conflicts, executor mode, and node availability.
func ValidateMultiArch(ctx context.Context, c client.Client, platforms []build.ImagePlatform, nodeSelector map[string]string, executor string) (bool, string, string) {
	if valid, reason, msg := ValidateMultiArchPlatforms(platforms); !valid {
		return valid, reason, msg
	}
	if valid, reason, msg := ValidateMultiArchNodeSelector(nodeSelector); !valid {
		return valid, reason, msg
	}
	if valid, reason, msg := ValidateMultiArchExecutor(executor); !valid {
		return valid, reason, msg
	}
	return ValidateMultiArchNodes(ctx, c, platforms)
}

// ValidateMultiArchNodes checks that the cluster has at least one schedulable
// node for each requested platform. It fetches all nodes once and builds a
// snapshot of available platforms, then checks the requested list against it.
func ValidateMultiArchNodes(ctx context.Context, c client.Client, platforms []build.ImagePlatform) (bool, string, string) {
	nodeList := &corev1.NodeList{}
	if err := c.List(ctx, nodeList); err != nil {
		return false, string(build.MultiArchNodeNotFound), fmt.Sprintf("failed to list nodes: %v", err)
	}

	available := availablePlatforms(nodeList.Items)

	for _, p := range platforms {
		key := p.OS + "/" + p.Arch
		if !available[key] {
			return false, string(build.MultiArchNodeNotFound), fmt.Sprintf(
				"no schedulable node found for platform %s", key)
		}
	}
	return true, "", ""
}

// availablePlatforms returns the set of os/arch combinations that have at
// least one schedulable, Ready node.
func availablePlatforms(nodes []corev1.Node) map[string]bool {
	platforms := make(map[string]bool)
	for _, node := range nodes {
		if node.Spec.Unschedulable {
			continue
		}
		ready := false
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				ready = true
				break
			}
		}
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

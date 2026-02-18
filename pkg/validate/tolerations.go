// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/ptr"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

// TolerationsRef contains all required fields
// to validate tolerations
type TolerationsRef struct {
	Build *build.Build // build instance for analysis
}

func NewTolerations(build *build.Build) *TolerationsRef {
	return &TolerationsRef{build}
}

// ValidatePath implements BuildPath interface and validates
// that tolerations key/operator/value are valid
func (b *TolerationsRef) ValidatePath(_ context.Context) error {
	ok, reason, msg := BuildRunTolerations(b.Build.Spec.Tolerations)
	if !ok {
		b.Build.Status.Reason = ptr.To(build.BuildReason(reason))
		b.Build.Status.Message = ptr.To(msg)
	}
	return nil
}

// BuildRunTolerations is used to validate tolerations in the BuildRun object
func BuildRunTolerations(tolerations []corev1.Toleration) (bool, string, string) {
	for _, toleration := range tolerations {
		// validate Key
		if errs := validation.IsQualifiedName(toleration.Key); errs != nil {
			return false, string(build.TolerationNotValid), fmt.Sprintf("Toleration key not valid: %v", strings.Join(errs, ", "))
		}
		// validate Operator
		if toleration.Operator != corev1.TolerationOpExists && toleration.Operator != corev1.TolerationOpEqual {
			return false, string(build.TolerationNotValid), fmt.Sprintf("Toleration operator not valid. Must be one of: '%v', '%v'", corev1.TolerationOpExists, corev1.TolerationOpEqual)
		}
		// validate Value
		if errs := validation.IsValidLabelValue(toleration.Value); errs != nil {
			return false, string(build.TolerationNotValid), fmt.Sprintf("Toleration value not valid: %v", strings.Join(errs, ", "))
		}
		// validate Taint Effect, of which only "NoSchedule" is supported
		if toleration.Effect != "" && toleration.Effect != corev1.TaintEffectNoSchedule {
			return false, string(build.TolerationNotValid), fmt.Sprintf("Only the '%v' toleration effect is supported.", corev1.TaintEffectNoSchedule)
		}
		// validate TolerationSeconds, which should not be specified
		if toleration.TolerationSeconds != nil {
			return false, string(build.TolerationNotValid), "Specifying TolerationSeconds is not supported."
		}
	}
	return true, "", ""
}

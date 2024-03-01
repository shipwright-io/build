// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

// SourcesRef implements RuntimeRef interface to add validations for `build.spec.source`.
type SourceRef struct {
	Build *build.Build // build instance for analysis
}

// ValidatePath executes the validation routine, inspecting the `build.spec.source` path
func (s *SourceRef) ValidatePath(_ context.Context) error {
	if s.Build.Spec.Source != nil {
		return s.validateSourceEntry(s.Build.Spec.Source)
	}

	return nil
}

// validateSourceEntry inspect informed entry, probes all required attributes.
func (s *SourceRef) validateSourceEntry(source *build.Source) error {

	// dont bail out if the Source object is empty, we preserve the old behaviour as in v1alpha1
	if source.Type == "" && source.Git == nil &&
		source.OCIArtifact == nil && source.Local == nil {
		return nil
	}

	switch source.Type {
	case "Git":
		if source.Git == nil || source.OCIArtifact != nil || source.Local != nil {
			return fmt.Errorf("type does not match the source")
		}
	case "OCI":
		if source.OCIArtifact == nil || source.Git != nil || source.Local != nil {
			return fmt.Errorf("type does not match the source")
		}
	case "Local":
		if source.Local == nil || source.OCIArtifact != nil || source.Git != nil {
			return fmt.Errorf("type does not match the source")
		}
	case "":
		return fmt.Errorf("type definition is missing")
	}
	return nil
}

// NewSourcesRef instantiate a new SourcesRef passing the build object pointer along.
func NewSourceRef(b *build.Build) *SourceRef {
	return &SourceRef{Build: b}
}

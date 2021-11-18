// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"
	"net/url"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

// SourcesRef implements RuntimeRef interface to add validations for `build.spec.sources` slice.
type SourcesRef struct {
	Build *build.Build // build instance for analysis
}

// ValidatePath executes the validation routine, inspecting the `build.spec.sources` path, which
// contains a slice of BuildSource.
func (s *SourcesRef) ValidatePath(_ context.Context) error {
	for _, source := range s.Build.Spec.Sources {
		if err := s.validateSourceEntry(source); err != nil {
			return err
		}
	}
	return nil
}

// validateSourceEntry inspect informed entry, probes all required attributes.
func (s *SourcesRef) validateSourceEntry(source build.BuildSource) error {
	if source.Name == "" {
		return fmt.Errorf("name must be informed")
	}
	if source.URL == "" {
		return fmt.Errorf("URL must be informed")
	}
	if _, err := url.ParseRequestURI(source.URL); err != nil {
		return err
	}
	return nil
}

// NewSourcesRef instantiate a new SourcesRef passing the build object pointer along.
func NewSourcesRef(b *build.Build) *SourcesRef {
	return &SourcesRef{Build: b}
}

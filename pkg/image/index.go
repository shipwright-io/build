// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// PlatformImageEntry associates a platform (os/arch) with a specific image reference.
type PlatformImageEntry struct {
	OS       string
	Arch     string
	ImageRef name.Reference
}

// AssembleImageIndex creates an OCI image index (manifest list) from a set of
// per-platform images. Each platform image is pulled from the registry, annotated
// with its platform descriptor, and appended to a new empty index.
func AssembleImageIndex(entries []PlatformImageEntry, options []remote.Option) (containerreg.ImageIndex, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("at least one platform image entry is required")
	}

	var idx containerreg.ImageIndex = empty.Index

	var addendums []mutate.IndexAddendum
	for _, entry := range entries {
		img, err := remote.Image(entry.ImageRef, options...)
		if err != nil {
			return nil, fmt.Errorf("pulling image for %s/%s from %s: %w", entry.OS, entry.Arch, entry.ImageRef.String(), err)
		}

		addendums = append(addendums, mutate.IndexAddendum{
			Add: img,
			Descriptor: containerreg.Descriptor{
				Platform: &containerreg.Platform{
					OS:           entry.OS,
					Architecture: entry.Arch,
				},
			},
		})
	}

	idx = mutate.AppendManifests(idx, addendums...)

	return idx, nil
}

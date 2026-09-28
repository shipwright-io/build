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
//
// If a platform reference points to an index instead of a single image, this function
// extracts the child image matching the requested platform. If no matching child is found,
// an error is returned.
func AssembleImageIndex(entries []PlatformImageEntry, options []remote.Option) (containerreg.ImageIndex, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("at least one platform image entry is required")
	}

	var idx containerreg.ImageIndex = empty.Index

	var addendums []mutate.IndexAddendum
	for _, entry := range entries {
		// Use remote.Get to retrieve the descriptor, which works for both images and indexes
		desc, err := remote.Get(entry.ImageRef, options...)
		if err != nil {
			return nil, fmt.Errorf("getting descriptor for %s/%s from %s: %w", entry.OS, entry.Arch, entry.ImageRef.String(), err)
		}

		var img containerreg.Image

		// Check if the descriptor points to an index or a single image
		if desc.MediaType.IsIndex() {
			// The platform reference points to an index; extract the child matching our platform
			idx, err := desc.ImageIndex()
			if err != nil {
				return nil, fmt.Errorf("parsing index for %s/%s from %s: %w", entry.OS, entry.Arch, entry.ImageRef.String(), err)
			}

			manifest, err := idx.IndexManifest()
			if err != nil {
				return nil, fmt.Errorf("reading index manifest for %s/%s from %s: %w", entry.OS, entry.Arch, entry.ImageRef.String(), err)
			}

			// Find the child matching the requested platform
			var matchedHash containerreg.Hash
			for _, m := range manifest.Manifests {
				if m.Platform != nil && m.Platform.OS == entry.OS && m.Platform.Architecture == entry.Arch {
					matchedHash = m.Digest
					break
				}
			}

			if matchedHash.String() == "" {
				return nil, fmt.Errorf("no child image found in index for platform %s/%s at %s", entry.OS, entry.Arch, entry.ImageRef.String())
			}

			// Retrieve the specific platform image by digest
			img, err = idx.Image(matchedHash)
			if err != nil {
				return nil, fmt.Errorf("extracting platform %s/%s image from index at %s: %w", entry.OS, entry.Arch, entry.ImageRef.String(), err)
			}
		} else {
			// It's a single image, use it directly
			img, err = desc.Image()
			if err != nil {
				return nil, fmt.Errorf("parsing image for %s/%s from %s: %w", entry.OS, entry.Arch, entry.ImageRef.String(), err)
			}
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

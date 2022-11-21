// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"fmt"
	"os"
	"path"

	"github.com/google/go-containerregistry/pkg/name"
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

// LoadImageOrImageIndexFromDirectory loads an image or an image index from a directory
func LoadImageOrImageIndexFromDirectory(directory string) (containerreg.Image, containerreg.ImageIndex, error) {
	// if we have an index.json, then we have an OCI image index
	fileInfo, err := os.Stat(path.Join(directory, "index.json"))
	if err == nil && !fileInfo.IsDir() {
		imageIndex, err := layout.ImageIndexFromPath(directory)
		if err != nil {
			return nil, nil, err
		}

		indexManifest, err := imageIndex.IndexManifest()
		if err != nil {
			return nil, nil, err
		}

		// flatten an index with exactly one entry, in particular if we push an index with an index that contains the images,
		// then docker pull will not work
	lengthOneManifestLoop:
		for len(indexManifest.Manifests) == 1 {
			desc := indexManifest.Manifests[0]

			switch {

			case desc.MediaType.IsImage():
				image, err := imageIndex.Image(desc.Digest)
				return image, nil, err

			case desc.MediaType.IsIndex():
				imageIndex, err = imageIndex.ImageIndex(desc.Digest)
				if err != nil {
					return nil, nil, err
				}

				indexManifest, err = imageIndex.IndexManifest()
				if err != nil {
					return nil, nil, err
				}

			default:
				break lengthOneManifestLoop
			}
		}

		return nil, imageIndex, nil
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, nil, err
	}

	if len(entries) == 1 {
		// tag nil is correct here, a tag would be needed if the tarball contains several images
		// which is not desired to be the case here
		image, err := tarball.ImageFromPath(path.Join(directory, entries[0].Name()), nil)
		return image, nil, err
	}
	return nil, nil, fmt.Errorf("no image was found at %q", directory)
}

// LoadImageOrImageIndexFromRegistry loads an image or an image index from a registry
func LoadImageOrImageIndexFromRegistry(imageName name.Reference, options []remote.Option) (containerreg.Image, containerreg.ImageIndex, error) {
	descriptor, err := remote.Head(imageName, options...)
	if err != nil {
		return nil, nil, err
	}

	if descriptor.MediaType.IsIndex() {
		imageIndex, err := remote.Index(imageName, options...)
		return nil, imageIndex, err
	}

	image, err := remote.Image(imageName, options...)
	return image, nil, err
}

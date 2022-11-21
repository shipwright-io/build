// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"github.com/google/go-containerregistry/pkg/name"
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// PushImageOrImageIndex pushes and image or image index and returns the digest and size. The size is only returned for an image.
func PushImageOrImageIndex(imageName name.Reference, image containerreg.Image, imageIndex containerreg.ImageIndex, options []remote.Option) (string, int64, error) {
	var digest string
	var size int64
	size = -1

	if image != nil {
		if err := remote.Write(imageName, image, options...); err != nil {
			return "", 0, err
		}

		hash, err := image.Digest()
		if err != nil {
			return "", 0, err
		}
		digest = hash.String()

		manifest, err := image.Manifest()
		if err != nil {
			return "", 0, err
		}

		size = manifest.Config.Size
		for _, layer := range manifest.Layers {
			size += layer.Size
		}
	}

	if imageIndex != nil {
		if err := remote.WriteIndex(imageName, imageIndex, options...); err != nil {
			return "", 0, err
		}

		hash, err := imageIndex.Digest()
		if err != nil {
			return "", 0, err
		}
		digest = hash.String()
	}

	return digest, size, nil
}

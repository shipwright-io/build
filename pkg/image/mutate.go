// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"errors"
	"time"

	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// replaceManifestIn replaces a manifest in an index, because there is no
// mutate.ReplaceManifests, so therefore, it removes the old one first,
// and then add the new one
func replaceManifestIn(imageIndex containerreg.ImageIndex, descriptor containerreg.Descriptor, replacement mutate.Appendable) containerreg.ImageIndex {
	imageIndex = mutate.RemoveManifests(imageIndex, func(ref containerreg.Descriptor) bool {
		return ref.Digest.String() == descriptor.Digest.String()
	})

	return mutate.AppendManifests(imageIndex, mutate.IndexAddendum{
		Add: replacement,
		Descriptor: containerreg.Descriptor{
			Annotations: descriptor.Annotations,
			MediaType:   descriptor.MediaType,
			Platform:    descriptor.Platform,
			URLs:        descriptor.URLs,
		},
	})
}

// MutateImageOrImageIndex mutates an image or image index with additional annotations and labels
func MutateImageOrImageIndex(image containerreg.Image, imageIndex containerreg.ImageIndex, annotations map[string]string, labels map[string]string) (containerreg.Image, containerreg.ImageIndex, error) {
	if imageIndex != nil {
		indexManifest, err := imageIndex.IndexManifest()
		if err != nil {
			return nil, nil, err
		}

		if len(labels) > 0 || len(annotations) > 0 {
			for _, descriptor := range indexManifest.Manifests {
				switch descriptor.MediaType {
				case types.OCIImageIndex, types.DockerManifestList:
					childImageIndex, err := imageIndex.ImageIndex(descriptor.Digest)
					if err != nil {
						return nil, nil, err
					}
					_, childImageIndex, err = MutateImageOrImageIndex(nil, childImageIndex, annotations, labels)
					if err != nil {
						return nil, nil, err
					}

					imageIndex = replaceManifestIn(imageIndex, descriptor, childImageIndex)

				case types.OCIManifestSchema1, types.DockerManifestSchema2:
					image, err := imageIndex.Image(descriptor.Digest)
					if err != nil {
						return nil, nil, err
					}

					image, err = mutateImage(image, annotations, labels)
					if err != nil {
						return nil, nil, err
					}

					imageIndex = replaceManifestIn(imageIndex, descriptor, image)
				}
			}
		}

		if len(annotations) > 0 {
			var castSucceeded bool
			imageIndex, castSucceeded = mutate.Annotations(imageIndex, annotations).(containerreg.ImageIndex)
			if !castSucceeded {
				return nil, nil, errors.New("expected mutate.Annotation to return an ImageIndex when passing in an ImageIndex")
			}
		}
	} else {
		var err error
		image, err = mutateImage(image, annotations, labels)
		if err != nil {
			return nil, nil, err
		}
	}

	return image, imageIndex, nil
}

func mutateImage(image containerreg.Image, annotations map[string]string, labels map[string]string) (containerreg.Image, error) {
	if len(labels) > 0 {
		cfg, err := image.ConfigFile()
		if err != nil {
			return nil, err
		}
		cfg = cfg.DeepCopy()

		if cfg.Config.Labels == nil {
			cfg.Config.Labels = labels
		} else {
			for key, value := range labels {
				cfg.Config.Labels[key] = value
			}
		}

		image, err = mutate.ConfigFile(image, cfg)
		if err != nil {
			return nil, err
		}
	}

	if len(annotations) > 0 {
		var castSucceeded bool
		image, castSucceeded = mutate.Annotations(image, annotations).(containerreg.Image)
		if !castSucceeded {
			return nil, errors.New("expected mutate.Annotation to return an Image when passing in an Image")
		}
	}

	return image, nil
}

func MutateImageOrImageIndexTimestamp(image containerreg.Image, imageIndex containerreg.ImageIndex, timestamp time.Time) (containerreg.Image, containerreg.ImageIndex, error) {
	if image != nil {
		image, err := mutateImageTimestamp(image, timestamp)
		return image, nil, err
	}

	imageIndex, err := mutateImageIndexTimestamp(imageIndex, timestamp)
	return nil, imageIndex, err
}

func mutateImageTimestamp(image containerreg.Image, timestamp time.Time) (containerreg.Image, error) {
	image, err := mutate.Time(image, timestamp)
	if err != nil {
		return nil, err
	}

	return image, nil
}

func mutateImageIndexTimestamp(imageIndex containerreg.ImageIndex, timestamp time.Time) (containerreg.ImageIndex, error) {
	indexManifest, err := imageIndex.IndexManifest()
	if err != nil {
		return nil, err
	}

	for _, desc := range indexManifest.Manifests {
		switch desc.MediaType {
		case types.OCIImageIndex, types.DockerManifestList:
			childImageIndex, err := imageIndex.ImageIndex(desc.Digest)
			if err != nil {
				return nil, err
			}

			childImageIndex, err = mutateImageIndexTimestamp(childImageIndex, timestamp)
			if err != nil {
				return nil, err
			}

			imageIndex = replaceManifestIn(imageIndex, desc, childImageIndex)

		case types.OCIManifestSchema1, types.DockerManifestSchema2:
			image, err := imageIndex.Image(desc.Digest)
			if err != nil {
				return nil, err
			}

			image, err = mutateImageTimestamp(image, timestamp)
			if err != nil {
				return nil, err
			}

			imageIndex = replaceManifestIn(imageIndex, desc, image)
		}
	}

	return imageIndex, nil
}

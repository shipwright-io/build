// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"errors"

	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// MutateImageOrImageIndex mutates an image or image index with additional annotations and labels
func MutateImageOrImageIndex(image containerreg.Image, imageIndex containerreg.ImageIndex, annotations map[string]string, labels map[string]string) (containerreg.Image, containerreg.ImageIndex, error) {
	if imageIndex != nil {
		indexManifest, err := imageIndex.IndexManifest()
		if err != nil {
			return nil, nil, err
		}

		if len(labels) > 0 || len(annotations) > 0 {
			for _, descriptor := range indexManifest.Manifests {
				digest := descriptor.Digest
				var appendable mutate.Appendable

				switch descriptor.MediaType {
				case types.OCIImageIndex, types.DockerManifestList:
					childImageIndex, err := imageIndex.ImageIndex(digest)
					if err != nil {
						return nil, nil, err
					}
					_, childImageIndex, err = MutateImageOrImageIndex(nil, childImageIndex, annotations, labels)
					if err != nil {
						return nil, nil, err
					}

					appendable = childImageIndex
				case types.OCIManifestSchema1, types.DockerManifestSchema2:
					image, err := imageIndex.Image(digest)
					if err != nil {
						return nil, nil, err
					}

					image, err = mutateImage(image, annotations, labels)
					if err != nil {
						return nil, nil, err
					}

					appendable = image
				default:
					continue
				}

				// there is no mutate.ReplaceManifests, therefore, remove the old one first, and then add the new one
				imageIndex = mutate.RemoveManifests(imageIndex, func(desc containerreg.Descriptor) bool {
					return desc.Digest.String() == digest.String()
				})

				imageIndex = mutate.AppendManifests(imageIndex, mutate.IndexAddendum{
					Add: appendable,
					Descriptor: containerreg.Descriptor{
						Annotations: descriptor.Annotations,
						MediaType:   descriptor.MediaType,
						Platform:    descriptor.Platform,
						URLs:        descriptor.URLs,
					},
				})
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

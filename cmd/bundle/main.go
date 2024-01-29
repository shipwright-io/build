// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/pflag"

	"github.com/shipwright-io/build/pkg/bundle"
	"github.com/shipwright-io/build/pkg/image"
)

type settings struct {
	help                      bool
	image                     string
	prune                     bool
	target                    string
	secretPath                string
	resultFileImageDigest     string
	resultFileSourceTimestamp string
}

var flagValues settings

func init() {
	// Explicitly define the help flag so that --help can be invoked and returns status code 0
	pflag.BoolVar(&flagValues.help, "help", false, "Print the help")

	// Main flags of the bundle step
	pflag.StringVar(&flagValues.image, "image", "", "Location of the bundle image (mandatory)")
	pflag.StringVar(&flagValues.target, "target", "/workspace/source", "The target directory to place the code")
	pflag.StringVar(&flagValues.resultFileImageDigest, "result-file-image-digest", "", "A file to write the image digest")
	pflag.StringVar(&flagValues.resultFileSourceTimestamp, "result-file-source-timestamp", "", "A file to write the source timestamp")

	pflag.StringVar(&flagValues.secretPath, "secret-path", "", "A directory that contains access credentials (optional)")
	pflag.BoolVar(&flagValues.prune, "prune", false, "Delete bundle image from registry after it was pulled")
}

func main() {
	if err := Do(context.Background()); err != nil {
		log.Fatal(err.Error())
	}
}

// Do is the main entry point of the bundle command
func Do(ctx context.Context) error {
	flagValues = settings{}
	pflag.Parse()

	if flagValues.help {
		pflag.Usage()
		return nil
	}

	if flagValues.image == "" {
		return fmt.Errorf("mandatory flag --image is not set")
	}

	ref, err := name.ParseReference(flagValues.image)
	if err != nil {
		return err
	}

	options, auth, err := image.GetOptions(ctx, ref, true, flagValues.secretPath, "Shipwright Build")
	if err != nil {
		return err
	}

	log.Printf("Pulling image %q", ref)
	desc, err := remote.Get(ref, options...)
	if err != nil {
		return err
	}

	img, err := desc.Image()
	if err != nil {
		return err
	}

	rc := mutate.Extract(img)
	defer rc.Close()

	unpackDetails, err := bundle.Unpack(rc, flagValues.target)
	if err != nil {
		return err
	}

	log.Printf("Image content was extracted to %s\n", flagValues.target)

	digest, err := img.Digest()
	if err != nil {
		return fmt.Errorf("failed to retrieve digest from bundle image: %w", err)
	}

	if flagValues.resultFileImageDigest != "" {
		if err = os.WriteFile(flagValues.resultFileImageDigest, []byte(digest.String()), 0644); err != nil {
			return err
		}
	}

	if flagValues.resultFileSourceTimestamp != "" {
		if unpackDetails.MostRecentFileTimestamp != nil {
			if err = os.WriteFile(flagValues.resultFileSourceTimestamp, []byte(strconv.FormatInt(unpackDetails.MostRecentFileTimestamp.Unix(), 10)), 0644); err != nil {
				return err
			}

		} else {
			log.Printf("Unable to determine source timestamp of content in %s\n", flagValues.target)
		}
	}

	if flagValues.prune {
		// Some container registry implementations, i.e. library/registry:2 will fail to
		// delete the image when there is no image digest given. Use image digest from the
		// image pulling to construct an image name including tag and digest.
		ref, err = name.NewDigest(fmt.Sprintf("%s@%s", ref.Name(), digest.String()))
		if err != nil {
			return err
		}

		log.Printf("Deleting image %q", ref)
		if err := image.Delete(ref, options, *auth); err != nil {
			return err
		}
	}

	return nil
}

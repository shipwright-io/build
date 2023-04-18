// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

// The algorithm to mutate the image was inspired by
// https://github.com/google/go-containerregistry/blob/main/cmd/crane/cmd/mutate.go

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/shipwright-io/build/pkg/image"
	"github.com/spf13/pflag"
)

// ExitError is an error which has an exit code to be used in os.Exit() to
// return both an exit code and an error message
type ExitError struct {
	Code    int
	Message string
	Cause   error
}

func (e ExitError) Error() string {
	return fmt.Sprintf("%s (exit code %d)", e.Message, e.Code)
}

type settings struct {
	help bool
	push string
	annotation,
	label *[]string
	insecure bool
	image,
	resultFileImageDigest,
	resultFileImageSize,
	secretPath string
}

func getAnnotation() []string {
	var annotation []string

	if flagValues.annotation != nil {
		return append(annotation, *flagValues.annotation...)
	}

	return annotation
}

func getLabel() []string {
	var label []string

	if flagValues.label != nil {
		return append(label, *flagValues.label...)
	}

	return label
}

var flagValues settings

func initializeFlag() {
	// Explicitly define the help flag so that --help can be invoked and returns status code 0
	pflag.BoolVar(&flagValues.help, "help", false, "Print the help")

	// Main flags for the image mutate step to define the configuration, for example
	// the flag `image` will always be used.
	pflag.StringVar(&flagValues.image, "image", "", "The name of image in container registry")
	pflag.StringVar(&flagValues.secretPath, "secret-path", "", "A directory that contains access credentials (optional)")
	pflag.BoolVar(&flagValues.insecure, "insecure", false, "Flag indicating the the container registry is insecure")

	pflag.StringVar(&flagValues.push, "push", "", "Push the image contained in this directory")

	flagValues.annotation = pflag.StringArray("annotation", nil, "New annotations to add")
	flagValues.label = pflag.StringArray("label", nil, "New labels to add")
	pflag.StringVar(&flagValues.resultFileImageDigest, "result-file-image-digest", "", "A file to write the image digest to")
	pflag.StringVar(&flagValues.resultFileImageSize, "result-file-image-size", "", "A file to write the image size to")
}

func main() {
	if err := Execute(context.Background()); err != nil {
		exitcode := 1

		switch err := err.(type) {
		case *ExitError:
			exitcode = err.Code
		}

		log.Print(err.Error())
		os.Exit(exitcode)
	}
}

// Execute performs flag parsing, input validation and the image mutation
func Execute(ctx context.Context) error {
	initializeFlag()
	pflag.Parse()

	if flagValues.help {
		pflag.Usage()
		return nil
	}

	return runImageProcessing(ctx)
}

func runImageProcessing(ctx context.Context) error {
	// parse the image name
	if flagValues.image == "" {
		return &ExitError{Code: 100, Message: "the 'image' argument must not be empty"}
	}
	imageName, err := name.ParseReference(flagValues.image)
	if err != nil {
		return fmt.Errorf("failed to parse image name: %w", err)
	}

	// parse annotations
	annotations, err := splitKeyVals(getAnnotation())
	if err != nil {
		return err
	}

	// parse labels
	labels, err := splitKeyVals(getLabel())
	if err != nil {
		return err
	}

	// prepare the registry options
	options, _, err := image.GetOptions(ctx, imageName, flagValues.insecure, flagValues.secretPath, "Shipwright Build")
	if err != nil {
		return err
	}

	// load the image or image index (usually multi-platform image)
	var img containerreg.Image
	var imageIndex containerreg.ImageIndex
	if flagValues.push == "" {
		log.Printf("Loading the image from the registry %q\n", imageName.String())
		img, imageIndex, err = image.LoadImageOrImageIndexFromRegistry(imageName, options)
	} else {
		log.Printf("Loading the image from the directory %q\n", flagValues.push)
		img, imageIndex, err = image.LoadImageOrImageIndexFromDirectory(flagValues.push)
	}
	if err != nil {
		log.Printf("Failed to load the image: %v\n", err)
		return err
	}
	if img != nil {
		log.Printf("Loaded single image")
	}
	if imageIndex != nil {
		log.Printf("Loaded image index")
	}

	// mutate the image
	if len(annotations) > 0 || len(labels) > 0 {
		log.Println("Mutating the image")
		img, imageIndex, err = image.MutateImageOrImageIndex(img, imageIndex, annotations, labels)
		if err != nil {
			log.Printf("Failed to mutate the image: %v\n", err)
			return err
		}
	}

	// push the image and determine the digest and size
	log.Printf("Pushing the image to registry %q\n", imageName.String())
	digest, size, err := image.PushImageOrImageIndex(imageName, img, imageIndex, options)
	if err != nil {
		log.Printf("Failed to push the image: %v\n", err)
		return err
	}

	// Writing image digest to file
	if digest != "" && flagValues.resultFileImageDigest != "" {
		if err := os.WriteFile(flagValues.resultFileImageDigest, []byte(digest), 0400); err != nil {
			return err
		}
	}

	// Writing image size in bytes to file
	if size > 0 && flagValues.resultFileImageSize != "" {
		if err := os.WriteFile(flagValues.resultFileImageSize, []byte(strconv.FormatInt(size, 10)), 0400); err != nil {
			return err
		}
	}

	return nil
}

// splitKeyVals splits key value pairs which is in form hello=world
func splitKeyVals(kvPairs []string) (map[string]string, error) {
	m := map[string]string{}

	for _, l := range kvPairs {
		parts := strings.SplitN(l, "=", 2)
		if len(parts) == 1 {
			return nil, fmt.Errorf("parsing label %q, not enough parts", l)
		}
		m[parts[0]] = parts[1]
	}

	return m, nil
}

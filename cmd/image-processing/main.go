// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

// The algorithm to mutate the image was inspired by
// https://github.com/google/go-containerregistry/blob/main/cmd/crane/cmd/mutate.go

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/image"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
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
	label []string
	insecure bool
	image,
	imageTimestamp,
	imageTimestampFile,
	resultFileImageDigest,
	resultFileImageSize,
	resultFileImageVulnerabilities,
	secretPath string
	vulnerabilitySettings   resources.VulnerablilityScanParams
	vulnerabilityCountLimit int
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

	pflag.StringArrayVar(&flagValues.annotation, "annotation", nil, "New annotations to add")
	pflag.StringArrayVar(&flagValues.label, "label", nil, "New labels to add")

	pflag.StringVar(&flagValues.imageTimestamp, "image-timestamp", "", "number to use as Unix timestamp to set image creation timestamp")
	pflag.StringVar(&flagValues.imageTimestampFile, "image-timestamp-file", "", "path to a file containing a unix timestamp to set as the image timestamp")

	pflag.StringVar(&flagValues.resultFileImageDigest, "result-file-image-digest", "", "A file to write the image digest to")
	pflag.StringVar(&flagValues.resultFileImageSize, "result-file-image-size", "", "A file to write the image size to")
	pflag.StringVar(&flagValues.resultFileImageVulnerabilities, "result-file-image-vulnerabilities", "", "A file to write the image vulnerabilities to")
	pflag.Var(&flagValues.vulnerabilitySettings, "vuln-settings", "Vulnerability settings json string. One can enable the scan by setting {\"enabled\":true} to this option")
	pflag.IntVar(&flagValues.vulnerabilityCountLimit, "vuln-count-limit", 50, "vulnerability count limit for the output of vulnerability scan")
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

	// validate that only one of the image timestamp flags are used
	if flagValues.imageTimestamp != "" && flagValues.imageTimestampFile != "" {
		pflag.Usage()
		return fmt.Errorf("image timestamp and image timestamp file flag is used, they are mutually exclusive, only use one")
	}

	// validate that image timestamp file exists (if set), and translate it into the imageTimestamp field
	if flagValues.imageTimestampFile != "" {
		_, err := os.Stat(flagValues.imageTimestampFile)
		if err != nil {
			return fmt.Errorf("image timestamp file flag references a non-existing file: %w", err)
		}

		data, err := os.ReadFile(flagValues.imageTimestampFile)
		if err != nil {
			return fmt.Errorf("failed to read image timestamp from %s: %w", flagValues.imageTimestampFile, err)
		}

		flagValues.imageTimestamp = string(data)
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
	annotations, err := splitKeyVals(flagValues.annotation)
	if err != nil {
		return err
	}

	// parse labels
	labels, err := splitKeyVals(flagValues.label)
	if err != nil {
		return err
	}

	// prepare the registry options
	options, auth, err := image.GetOptions(ctx, imageName, flagValues.insecure, flagValues.secretPath, "Shipwright Build")
	if err != nil {
		return err
	}

	// load the image or image index (usually multi-platform image)
	var img containerreg.Image
	var imageIndex containerreg.ImageIndex
	var isImageFromTar bool
	if flagValues.push == "" {
		log.Printf("Loading the image from the registry %q\n", imageName.String())
		img, imageIndex, err = image.LoadImageOrImageIndexFromRegistry(imageName, options)
	} else {
		log.Printf("Loading the image from the directory %q\n", flagValues.push)
		img, imageIndex, isImageFromTar, err = image.LoadImageOrImageIndexFromDirectory(flagValues.push)
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

	// check for image vulnerabilities if vulnerability scanning is enabled.
	var vulns []buildapi.Vulnerability

	if flagValues.vulnerabilitySettings.Enabled {
		var imageString string
		var imageInDir bool
		if flagValues.push != "" {
			imageString = flagValues.push

			// for single image in a tar file
			if isImageFromTar {
				entries, err := os.ReadDir(flagValues.push)
				if err != nil {
					return err
				}
				imageString = filepath.Join(imageString, entries[0].Name())
			}
			imageInDir = true
		} else {
			imageString = imageName.String()
			imageInDir = false
		}
		vulns, err = image.RunVulnerabilityScan(ctx, imageString, flagValues.vulnerabilitySettings.VulnerabilityScanOptions, auth, flagValues.insecure, imageInDir, flagValues.vulnerabilityCountLimit)
		if err != nil {
			return err
		}

		// log all the vulnerabilities
		if len(vulns) > 0 {
			log.Println("vulnerabilities found in the output image :")
			for _, vuln := range vulns {
				log.Printf("ID: %s, Severity: %s\n", vuln.ID, vuln.Severity)
			}
		}
		vulnOuput := serializeVulnerabilities(vulns)
		if err := os.WriteFile(flagValues.resultFileImageVulnerabilities, vulnOuput, 0640); err != nil {
			return err
		}
	}

	// Don't push the image if fail is set to true for shipwright managed push
	if flagValues.push != "" {
		if flagValues.vulnerabilitySettings.FailOnFinding && len(vulns) > 0 {
			log.Println("vulnerabilities have been found in the output image, exiting with code 22")
			return &ExitError{Code: 22, Message: "vulnerabilities found, exiting with code 22", Cause: errors.New("vulnerabilities found in the image")}
		}
	}
	// mutate the image timestamp
	if flagValues.imageTimestamp != "" {
		sec, err := strconv.ParseInt(flagValues.imageTimestamp, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse image timestamp value %q as a number: %w", flagValues.imageTimestamp, err)
		}

		log.Println("Mutating the image timestamp")
		img, imageIndex, err = image.MutateImageOrImageIndexTimestamp(img, imageIndex, time.Unix(sec, 0))
		if err != nil {
			return fmt.Errorf("failed to mutate the timestamp: %w", err)
		}
	}

	// push the image and determine the digest and size
	log.Printf("Pushing the image to registry %q\n", imageName.String())
	digest, size, err := image.PushImageOrImageIndex(imageName, img, imageIndex, options)
	if err != nil {
		log.Printf("Failed to push the image: %v\n", err)
		return err
	}

	log.Printf("Image %s@%s pushed\n", imageName.String(), digest)

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

func serializeVulnerabilities(Vulnerabilities []buildapi.Vulnerability) []byte {
	var output []string
	for _, vuln := range Vulnerabilities {
		output = append(output, fmt.Sprintf("%s:%c", vuln.ID, vuln.Severity[0]))
	}
	return []byte(strings.Join(output, ","))
}

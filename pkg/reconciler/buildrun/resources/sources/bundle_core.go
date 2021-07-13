// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"archive/tar"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

const (
	shpIgnoreFilename = ".shpignore"
	bundleBase        = "/"
)

// Bundle a local directory as-is into a container image. See remote.Option
// for optional options to the image push to the registry, for example to
// provide the appropriate access credentials.
func Bundle(ref name.Reference, directory string, options ...remote.Option) (name.Digest, error) {
	image, err := createBundleImage(directory)
	if err != nil {
		return name.Digest{}, err
	}

	hash, err := image.Digest()
	if err != nil {
		return name.Digest{}, err
	}

	if err := remote.Write(ref, image, options...); err != nil {
		return name.Digest{}, err
	}

	return name.NewDigest(fmt.Sprintf("%s@%v",
		ref.Name(),
		hash.String(),
	))
}

func createBundleImage(directory string) (v1.Image, error) {
	bundleLayer, err := bundleDirectory(directory)
	if err != nil {
		return nil, err
	}

	image, err := mutate.AppendLayers(empty.Image, bundleLayer)
	if err != nil {
		return nil, err
	}

	return mutate.Time(image, time.Unix(0, 0))
}

func bundleDirectory(directory string) (v1.Layer, error) {
	var split = func(path string) []string { return strings.Split(path, string(filepath.Separator)) }

	var patterns []gitignore.Pattern
	if file, err := os.Open(filepath.Join(directory, shpIgnoreFilename)); err == nil {
		defer file.Close()

		domain := split(directory)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if len(line) != 0 && !strings.HasPrefix(line, "#") {
				patterns = append(patterns, gitignore.ParsePattern(line, domain))
			}
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	matcher := gitignore.NewMatcher(patterns)

	var buf bytes.Buffer
	var tw = tar.NewWriter(&buf)
	defer tw.Close()

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		// Bail out on path errors
		if err != nil {
			return err
		}

		if matcher.Match(split(path), info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		inBundlePath, err := tarPath(directory, path)
		if err != nil {
			return err
		}

		switch {
		case info.Mode().IsDir():
			return tw.WriteHeader(&tar.Header{
				Name:     inBundlePath,
				Typeflag: tar.TypeDir,
				Mode:     int64(info.Mode()),
			})

		case info.Mode().IsRegular():
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			defer file.Close()

			if err := tw.WriteHeader(&tar.Header{
				Name:     inBundlePath,
				Typeflag: tar.TypeReg,
				Mode:     int64(info.Mode()),
				Size:     info.Size()}); err != nil {
				return err
			}

			_, err = io.Copy(tw, file)
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return tarball.LayerFromReader(&buf)
}

func tarPath(baseDir string, path string) (string, error) {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return "", err
	}

	return filepath.Join(bundleBase, rel), nil
}

// Unbundle a container image layer content into a local directory. Analog to
// the sources.Bundle function, optional remote.Option can be used to configure
// settings for the image pull, i.e. access credentials.
func Unbundle(ref name.Reference, targetPath string, options ...remote.Option) error {
	desc, err := remote.Get(ref, options...)
	if err != nil {
		return err
	}

	image, err := desc.Image()
	if err != nil {
		return err
	}

	rc := mutate.Extract(image)
	defer rc.Close()

	var tr = tar.NewReader(rc)
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil

		case err != nil:
			return err

		case header == nil:
			continue
		}

		var target = filepath.Join(targetPath, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := createDirectory(target); err != nil {
				return err
			}

		case tar.TypeReg:
			dir, _ := filepath.Split(target)
			if err := createDirectory(dir); err != nil {
				return err
			}

			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(file, tr); err != nil {
				return err
			}

			file.Close()
		}
	}
}

func createDirectory(path string) error {
	if _, err := os.Stat(path); err != nil {
		return os.MkdirAll(path, 0755)
	}

	return nil
}

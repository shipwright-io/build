// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"archive/tar"
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/google/go-containerregistry/pkg/name"
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const shpIgnoreFilename = ".shpignore"

// orasUnpackAnnotation is set by ORAS on layers that hold a directory packed as
// a gzipped tarball, as opposed to a single file stored as is.
const orasUnpackAnnotation = "io.deis.oras.content.unpack"

// UnpackDetails contains details about the files that were unpacked
type UnpackDetails struct {
	MostRecentFileTimestamp *time.Time
}

// PackAndPush a local directory as-is into a container image. See
// remote.Option for optional options to the image push to the registry, for
// example to provide the appropriate access credentials.
func PackAndPush(ref name.Reference, directory string, options ...remote.Option) (name.Digest, error) {
	bundleLayer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) { return Pack(directory) })
	if err != nil {
		return name.Digest{}, err
	}

	image, err := mutate.Time(empty.Image, time.Unix(0, 0))
	if err != nil {
		return name.Digest{}, err
	}

	image, err = mutate.AppendLayers(image, bundleLayer)
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

// PullAndUnpack a container image layer content into a local directory. Analog
// to the bundle.PackAndPush function, optional remote.Option can be used to
// configure settings for the image pull, i.e. access credentials.
func PullAndUnpack(ref name.Reference, targetPath string, options ...remote.Option) (containerreg.Image, error) {
	desc, err := remote.Get(ref, options...)
	if err != nil {
		return nil, err
	}

	image, err := desc.Image()
	if err != nil {
		return nil, err
	}

	if _, err = UnpackImage(image, targetPath); err != nil {
		return nil, err
	}

	return image, nil
}

// UnpackImage writes the content of an image into the local file system. A
// regular container image is flattened and its file system is unpacked. An
// artifact pushed by ORAS, where layers carry an org.opencontainers.image.title
// annotation, is unpacked layer by layer instead: a directory layer is unpacked
// as a tarball, and a file layer is written as is under its title.
func UnpackImage(image containerreg.Image, targetPath string) (*UnpackDetails, error) {
	manifest, err := image.Manifest()
	if err != nil {
		return nil, err
	}

	if !isORASArtifact(manifest) {
		rc := mutate.Extract(image)
		defer rc.Close()

		return Unpack(rc, targetPath)
	}

	layers, err := image.Layers()
	if err != nil {
		return nil, err
	}

	if len(layers) != len(manifest.Layers) {
		return nil, fmt.Errorf("image has %d layers, but its manifest lists %d", len(layers), len(manifest.Layers))
	}

	var details = UnpackDetails{}
	for i, layer := range layers {
		annotations := manifest.Layers[i].Annotations

		var layerDetails *UnpackDetails
		if annotations[orasUnpackAnnotation] == "true" {
			layerDetails, err = unpackLayer(layer, targetPath)
		} else {
			err = writeLayerFile(layer, targetPath, annotations[ocispec.AnnotationTitle])
		}
		if err != nil {
			return nil, err
		}

		if layerDetails != nil && layerDetails.MostRecentFileTimestamp != nil &&
			(details.MostRecentFileTimestamp == nil || details.MostRecentFileTimestamp.Before(*layerDetails.MostRecentFileTimestamp)) {
			details.MostRecentFileTimestamp = layerDetails.MostRecentFileTimestamp
		}
	}

	// Files stored as is carry no timestamp, so fall back to the creation time of the artifact
	if details.MostRecentFileTimestamp == nil {
		if created, err := time.Parse(time.RFC3339, manifest.Annotations[ocispec.AnnotationCreated]); err == nil {
			details.MostRecentFileTimestamp = &created
		}
	}

	return &details, nil
}

func isORASArtifact(manifest *containerreg.Manifest) bool {
	for _, layer := range manifest.Layers {
		if layer.Annotations[ocispec.AnnotationTitle] != "" {
			return true
		}
	}

	return false
}

func unpackLayer(layer containerreg.Layer, targetPath string) (*UnpackDetails, error) {
	rc, err := layer.Uncompressed()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return Unpack(rc, targetPath)
}

func writeLayerFile(layer containerreg.Layer, targetPath string, title string) error {
	if title == "" {
		return fmt.Errorf("layer has no %s annotation, unable to determine its file name", ocispec.AnnotationTitle)
	}

	// #nosec G305 path traversal is checked by validating that the resulting path does not escape targetPath
	var target = filepath.Join(targetPath, title)
	if rel, err := filepath.Rel(targetPath, target); err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("targetPath validation failed, %q escapes the target directory", title)
	}

	if err := os.MkdirAll(filepath.Dir(target), os.FileMode(0755)); err != nil {
		return err
	}

	// The blob of a file layer is the file content itself, ORAS does not compress it
	rc, err := layer.Compressed()
	if err != nil {
		return err
	}
	defer rc.Close()

	// #nosec G304 the target path was validated above
	file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(0644))
	if err != nil {
		return err
	}

	if _, err := io.Copy(file, rc); err != nil {
		_ = file.Close()
		return err
	}

	return file.Close()
}

// Pack reads a directory and creates a tar stream with its content by:
// - storing all directories and regular files as-is,
// - dereferencing all symlinks and storing the respective target,
// - ignoring all files configured in .shpignore
func Pack(directory string) (io.ReadCloser, error) {
	var split = func(path string) []string { return strings.Split(path, string(filepath.Separator)) }

	var write = func(w io.Writer, path string) error {
		// #nosec G304 names are safe, they come from the listing
		file, err := os.Open(path)
		if err != nil {
			return err
		}

		defer file.Close()

		_, err = io.Copy(w, file)
		return err
	}

	var followSymLink = func(path string) (string, os.FileInfo, error) {
		deref, err := os.Readlink(path)
		if err != nil {
			return "", nil, err
		}

		if !filepath.IsAbs(deref) {
			deref = filepath.Join(
				filepath.Dir(path),
				deref,
			)
		}

		info, err := os.Stat(deref)
		return deref, info, err
	}

	var patterns []gitignore.Pattern
	// #nosec G304 names are safe
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

	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	var tw = tar.NewWriter(w)
	defer func() {
		_ = tw.Close()
		_ = w.Close()
	}()

	err = filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		// Bail out on path errors
		if err != nil {
			return err
		}

		// Skip files on the ignore list
		if matcher.Match(split(path), d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, path)
		if err != nil {
			return err
		}

		header.Name, err = filepath.Rel(directory, path)
		if err != nil {
			return err
		}

		switch {
		case info.Mode().IsDir():
			return tw.WriteHeader(header)

		case info.Mode().IsRegular():
			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			return write(tw, path)

		case info.Mode()&os.ModeSymlink == os.ModeSymlink:
			deref, info, err := followSymLink(path)
			if err != nil {
				return err
			}

			header, err = tar.FileInfoHeader(info, deref)
			if err != nil {
				return err
			}

			header.Name, err = filepath.Rel(directory, path)
			if err != nil {
				return err
			}

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			return write(tw, deref)

		default:
			return fmt.Errorf("unsupported file type: %s", path)
		}
	})

	return r, err
}

// Unpack reads a tar stream and writes the content into the local file system
// with all files and directories.
func Unpack(in io.Reader, targetPath string) (*UnpackDetails, error) {
	type chmod struct {
		name string
		mode os.FileMode
	}

	// Make sure the target path exists and is a directory
	if stat, err := os.Stat(targetPath); err != nil {
		if err := os.MkdirAll(targetPath, os.FileMode(0755)); err != nil {
			return nil, err
		}
	} else if !stat.IsDir() {
		return nil, fmt.Errorf("target %q exists, but it's not a directory", targetPath)
	}

	var chmods []chmod
	var details = UnpackDetails{}
	var tr = tar.NewReader(in)
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			// before leaving, make sure to set the file permissions to the ones specified in the tar stream
			for _, chmod := range chmods {
				if err := os.Chmod(chmod.name, chmod.mode); err != nil {
					return nil, err
				}
			}

			return &details, nil

		case err != nil:
			return nil, err

		case header == nil:
			continue
		}

		// #nosec G305 path traversal is checked by validating that the resulting path does not escape targetPath
		var target = filepath.Join(targetPath, header.Name)
		if rel, err := filepath.Rel(targetPath, target); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("targetPath validation failed, %q escapes the target directory", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Skip the root directory, since it already exists
			if target == targetPath {
				continue
			}

			if err := os.MkdirAll(target, os.FileMode(0777)); err != nil {
				return nil, err
			}

			chmods = append(chmods, chmod{name: target, mode: fileMode(header)})

		case tar.TypeReg:
			// Edge case in which that tarball did not have a directory entry
			dir, _ := filepath.Split(target)
			if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
				return nil, err
			}

			// #nosec G304 names are safe, they come from the listing
			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, fileMode(header))
			if err != nil {
				return nil, err
			}

			if _, err := io.Copy(file, tr); err != nil {
				_ = file.Close()
				return nil, err
			}

			if err := file.Close(); err != nil {
				return nil, err
			}

			if err := os.Chtimes(target, header.AccessTime, header.ModTime); err != nil {
				return nil, err
			}

			if details.MostRecentFileTimestamp == nil || details.MostRecentFileTimestamp.Before(header.ModTime) {
				details.MostRecentFileTimestamp = &header.ModTime
			}

		default:
			return nil, fmt.Errorf("provided tarball contains unsupported file type, only directories and regular files are supported")
		}
	}
}

func fileMode(tarHeader *tar.Header) os.FileMode {
	mode := tarHeader.Mode
	if mode < 0 || mode > math.MaxUint32 {
		return 0
	}

	// #nosec G115 was checked above
	return os.FileMode(mode)
}

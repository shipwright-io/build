// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

// BundleSourceDirectory packages a directory's contents into a single-layer OCI image
// suitable for distributing source code as an OCI artifact. The directory is archived
// as a gzipped tar and appended as a layer to a scratch image.
func BundleSourceDirectory(dir string) (containerreg.Image, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("source directory does not exist: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	var buf bytes.Buffer
	if err := createTarGz(&buf, dir); err != nil {
		return nil, fmt.Errorf("creating source tarball: %w", err)
	}

	compressed := buf.Bytes()
	opener := func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(compressed)), nil
	}

	layer, err := tarball.LayerFromOpener(opener)
	if err != nil {
		return nil, fmt.Errorf("creating layer from tarball: %w", err)
	}

	img, err := mutate.AppendLayers(empty.Image, layer)
	if err != nil {
		return nil, fmt.Errorf("appending layer to empty image: %w", err)
	}

	return img, nil
}

func createTarGz(w io.Writer, srcDir string) error {
	gzWriter := gzip.NewWriter(w)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return filepath.Walk(srcDir, func(filePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(srcDir, filePath)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		// Dereference symlinks so the tar only contains TypeDir and TypeReg
		// entries, which is required for compatibility with bundle.Unpack().
		if info.Mode()&os.ModeSymlink != 0 {
			info, err = os.Stat(filePath)
			if err != nil {
				return fmt.Errorf("dereferencing symlink %s: %w", filePath, err)
			}
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("creating tar header for %s: %w", relPath, err)
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("writing tar header for %s: %w", relPath, err)
		}

		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}

		f, err := os.Open(filePath) // #nosec G304 G122 -- filePath is constructed from filepath.Walk on a controller-owned workspace
		if err != nil {
			return fmt.Errorf("opening %s: %w", filePath, err)
		}
		defer f.Close()

		if _, err := io.Copy(tarWriter, f); err != nil {
			return fmt.Errorf("writing %s to tar: %w", relPath, err)
		}

		return nil
	})
}

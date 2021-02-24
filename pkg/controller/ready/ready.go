// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package ready

import "os"

// Ready holds state about whether the controller is ready and communicates
// that to a Kubernetes readiness/liveness probe.
type Ready interface {
	// Set ensures that future readiness/liveness probes will indicate that
	// the controller is ready.
	Set() error

	// Unset ensures that future readiness/liveness probes will indicate
	// that the controller is not ready.
	Unset() error
}

type fileReady struct {
	filename string
}

// NewFileReady returns a Ready that uses the presence of a file on disk to
// communicate whether the controller is ready.
func NewFileReady(filename string) Ready {
	return fileReady{filename}
}

// Set creates a file on disk whose presence can be used by a
// readiness/liveness probe.
func (r fileReady) Set() error {
	f, err := os.Create(r.filename)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}

	return f.Close()
}

// Unset removes the file on disk that was created by Set().
func (r fileReady) Unset() error {
	if _, err := os.Stat(r.filename); os.IsNotExist(err) {
		return nil
	}

	return os.Remove(r.filename)
}

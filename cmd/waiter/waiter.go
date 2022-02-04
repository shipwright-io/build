// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// Waiter represents the actor that will wait for timeout, using a lock-file to keep it actively
// waiting. When "done" is issued the lock-file is removed and the waiter ends.
type Waiter struct {
	flagValues *settings // command-line flags
}

// ErrTimeout emitted when timeout is reached.
var ErrTimeout = errors.New("timeout waiting for condition")

// save writes the lock-file with informed PID.
func (w *Waiter) save(pid int) error {
	return os.WriteFile(w.flagValues.lockFile, []byte(strconv.Itoa(pid)), 0600)
}

// read reads the lock-file, must contain an integer.
func (w *Waiter) read() (int, error) {
	data, err := os.ReadFile(w.flagValues.lockFile)
	if err != nil {
		return -1, err
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return -1, err
	}
	return pid, nil
}

// validate that a lock file not longer exists, otherwise timeout if needed
func (w *Waiter) retry() error {
	timer := time.NewTimer(w.flagValues.timeout)
	defer timer.Stop()

	// Verify on file existence every 100ms
	ticker := time.Tick(100 * time.Millisecond)

	for {
		select {
		case <-timer.C:
			return fmt.Errorf("%w: elapsed %v seconds", ErrTimeout, w.flagValues.timeout.Seconds())
		case <-ticker:
			if _, err := os.Stat(w.flagValues.lockFile); err != nil && os.IsNotExist(err) {
				log.Printf("Done! Condition has been reached\n")
				return nil
			}
			// do nothing and continue the ticker
		}
	}

}

// Wait wait for the lock-file to be removed, or timeout.
func (w *Waiter) Wait() error {
	pid := os.Getpid()
	if err := w.save(pid); err != nil {
		return err
	}

	// waiting for the lock-file removal...
	err := w.retry()
	if err != nil {
		_ = os.RemoveAll(w.flagValues.lockFile)
	}
	return err
}

// Done removes the lock-file.
func (w *Waiter) Done() error {
	pid, err := w.read()
	if err != nil {
		return err
	}
	log.Printf("Removing lock-file at '%s' (%d PID)", w.flagValues.lockFile, pid)
	return os.Remove(w.flagValues.lockFile)
}

// NewWaiter instantiate a new waiter, making sure the timeout informed is acceptable.
func NewWaiter(flagValues settings) *Waiter {
	if flagValues.timeout <= time.Second {
		log.Printf("Warning! The timeout informed '%s' is lower than 1s!\n", flagValues.timeout)
		flagValues.timeout = defaultTimeout
	}
	return &Waiter{flagValues: &flagValues}
}

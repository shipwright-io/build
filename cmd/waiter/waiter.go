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

// retry re-execute the informed function waiting for 100ms per attempt.
func retry(timeout time.Duration, fn func() bool) error {
	attempts := int(int(timeout.Milliseconds()) / 100)
	log.Printf("Will retry '%d' times (sleep 100ms)...\n", attempts)
	for i := attempts; i > 0; i-- {
		if fn() {
			log.Printf("Done! Condition has been reached on '%d' attempt\n", attempts-i)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("%w: elapsed %v seconds", ErrTimeout, timeout.Seconds())
}

// Wait wait for the lock-file to be removed, or timeout.
func (w *Waiter) Wait() error {
	pid := os.Getpid()
	if err := w.save(pid); err != nil {
		return err
	}

	// waiting for the lock-file removal...
	err := retry(w.flagValues.timeout, func() bool {
		_, err := os.Stat(w.flagValues.lockFile)
		return err != nil && os.IsNotExist(err)
	})
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

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// settings composed by command-line flag values.
type settings struct {
	lockFile string        // path to lock file
	timeout  time.Duration // how long wait for 'done'
}

const longDesc = `
# waiter

Idle loop to hold a container (possibly a Kubernetes POD) running while some other
action happens in the background. It is started by issuing "waiter start" and can
be stopped with "waiter done", or after timeout.

## Usage

Start the waiting, use --timeout to change how long:

	$ waiter start

You can signal "done" by running:

	$ waiter done

Or, alternatively:

	$ rm -f <lock-file>

## Return-Code

In the case of timeout, the waiter will return error, it only exits gracefully via
"waiter done", or the removal of the lock-file (before timeout).
`

var (
	rootCmd  = newRootCmd()
	startCmd = newStartCmd()
	doneCmd  = newDoneCmd()
)

// defaultTimeout default timeout duration.
var defaultTimeout = 60 * time.Second

// defaultLockFile default location of the lock-file.
var defaultLockFile = "/tmp/waiter.lock"

// flagValues receives the command-line flag values.
var flagValues = settings{}

// init assembles the flags and the cobra sub-commands.
func init() {
	flags := rootCmd.PersistentFlags()

	flags.StringVar(&flagValues.lockFile, "lock-file", defaultLockFile, "lock file full path")
	flags.DurationVar(&flagValues.timeout, "timeout", defaultTimeout, "how long to wait until 'done'")

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(doneCmd)
}

func newRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "waiter [flags]",
		Short: "Will wait until `done` issued",
		Long:  longDesc,
	}
}

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "start",
		Short:        "Starts the wait, and holds until `done` is issued.",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			w := NewWaiter(flagValues)
			return w.Wait()
		},
	}
}

func newDoneCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "done",
		Aliases:      []string{"stop"},
		Short:        "Interrupts the waiting.",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			w := NewWaiter(flagValues)
			return w.Done()
		},
	}
}

// main waiter's entrypoint.
func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("[ERROR] %v\n", err)
	}
	os.Exit(0)
}

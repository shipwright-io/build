// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"os"
	"os/exec"
	"time"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// executable path to the waiter executable file.
var executable string

// building the command-line application before starting the test suite, it will populate the
// global variable with the path to the waiter binary compiled.
var _ = BeforeSuite(func() {
	var err error
	executable, err = gexec.Build("github.com/shipwright-io/build/cmd/waiter")
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("Waiter", func() {
	// run creates a exec.Command instance using the arguments informed.
	var run = func(args ...string) *gexec.Session {
		cmd := exec.Command(executable)
		cmd.Args = append(cmd.Args, args...)
		stdin := &bytes.Buffer{}
		cmd.Stdin = stdin

		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		// when "start" sub-command is issued, a graceful wait takes place for the asynchronous
		// instantiation of the command-line application and creation of the lock-file
		for _, arg := range args {
			if arg == "start" {
				time.Sleep(3 * time.Second)
			}
		}
		return session
	}

	// inspectSession inspect the informed session to identify if the informed matcher is true,
	// after inspection it closes the informed channel.
	var inspectSession = func(
		session *gexec.Session,
		doneCh chan interface{},
		matcher types.GomegaMatcher,
	) {
		defer GinkgoRecover()

		Eventually(session, defaultTimeout).Should(matcher)
		close(doneCh)
	}

	When("--help is passed", func() {
		var session *gexec.Session

		BeforeEach(func() {
			session = run("--help")
		})

		It("shows the general help message", func() {
			Eventually(session).Should(gbytes.Say("Usage:"))
		})
	})

	Describe("expect to succeed when lock-file removed before timeout", func() {
		var startCh = make(chan interface{})

		BeforeEach(func() {
			session := run("start")

			go inspectSession(session, startCh, gexec.Exit(0))
		})

		It("stops when lock-file is removed", func() {
			err := os.RemoveAll(defaultLockFile)
			Expect(err).ToNot(HaveOccurred())

			Eventually(startCh, defaultTimeout).Should(BeClosed())
		})
	})

	Describe("expect to succeed when `done` is issued before timeout", func() {
		var startCh = make(chan interface{})
		var doneCh = make(chan interface{})

		BeforeEach(func() {
			session := run("start")

			go inspectSession(session, startCh, gexec.Exit(0))
		})

		BeforeEach(func() {
			session := run("done")

			go inspectSession(session, doneCh, gexec.Exit(0))
		})

		It("stops when done is issued", func() {
			Eventually(startCh, defaultTimeout).Should(BeClosed())
			Eventually(doneCh, defaultTimeout).Should(BeClosed())
		})
	})

	Describe("expect to fail when timeout is reached", func() {
		var startCh = make(chan interface{})

		BeforeEach(func() {
			session := run("start", "--timeout", "2s")

			go inspectSession(session, startCh, gexec.Exit(1))
		})

		It("stops when timeout is reached", func() {
			Eventually(startCh, defaultTimeout).Should(BeClosed())
		})
	})
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	_ = os.RemoveAll(defaultLockFile)
})

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/shipwright-io/build/pkg/util"
)

var _ = Describe("TCP", func() {

	Context("TestConnection", func() {

		var result bool
		var hostname string
		var port int

		JustBeforeEach(func() {
			result = util.TestConnection(hostname, port, 1)
		})

		Context("For a broken endpoint", func() {

			BeforeEach(func() {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				Expect(err).ToNot(HaveOccurred())

				tcpAddress := listener.Addr().(*net.TCPAddr)
				hostname = tcpAddress.IP.String()
				port = tcpAddress.Port

				Expect(listener.Close()).To(Succeed())
			})

			It("returns false", func() {
				Expect(result).To(BeFalse())
			})
		})

		Context("For a functional endpoint", func() {

			BeforeEach(func() {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(listener.Close)

				tcpAddress := listener.Addr().(*net.TCPAddr)
				hostname = tcpAddress.IP.String()
				port = tcpAddress.Port
			})

			It("returns true", func() {
				Expect(result).To(BeTrue())
			})
		})
	})
})

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
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
				hostname = "shipwright.io"
				port = 33333
			})

			It("returns false", func() {
				Expect(result).To(BeFalse())
			})
		})

		Context("For an unknown host", func() {

			BeforeEach(func() {
				hostname = "shipwright-dhasldglidgewidgwd.io"
				port = 33333
			})

			It("returns false", func() {
				Expect(result).To(BeFalse())
			})
		})

		Context("For a functional endpoint", func() {

			BeforeEach(func() {
				hostname = "github.com"
				port = 443
			})

			It("returns true", func() {
				Expect(result).To(BeTrue())
			})
		})
	})
})

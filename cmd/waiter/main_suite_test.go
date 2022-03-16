// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWaiterCmd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Waiter")
}

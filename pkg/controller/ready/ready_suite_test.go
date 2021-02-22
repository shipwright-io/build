// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package ready_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestReady(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ready Suite")
}

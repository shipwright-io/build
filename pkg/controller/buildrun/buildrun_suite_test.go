// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBuildRun(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BuildRun Suite")
}

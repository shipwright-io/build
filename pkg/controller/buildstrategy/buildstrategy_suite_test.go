// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildstrategy_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBuild(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BuildStrategy Suite")
}

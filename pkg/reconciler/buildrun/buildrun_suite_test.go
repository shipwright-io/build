// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBuildrun(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Buildrun Suite")
}

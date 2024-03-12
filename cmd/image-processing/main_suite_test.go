// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

func TestImageProcessingCmd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Image Processing Command Suite")
}

func FailWith(substr string) types.GomegaMatcher {
	return MatchError(ContainSubstring(substr))
}

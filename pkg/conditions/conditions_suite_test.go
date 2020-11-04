// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package conditions_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConditions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Conditions Suite")
}

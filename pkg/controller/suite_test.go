// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package controller_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controllers Suite")
}

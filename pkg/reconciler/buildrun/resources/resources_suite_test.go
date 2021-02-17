// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package resources_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestResources(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resources Suite")
}

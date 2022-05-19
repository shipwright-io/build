// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package volumes_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestImage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Volumes Suite")
}

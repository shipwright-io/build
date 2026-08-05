// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	_ "github.com/shipwright-io/build/pkg/config"
)

func TestValidate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validate Suite")
}
